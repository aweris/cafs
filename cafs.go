package cafs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/aweris/cafs/internal/remote"
)

const digestPrefix = "sha256:"

// CAS is the main content-addressable storage implementation.
type CAS struct {
	blobs    *blobStore
	entries  sync.Map // key -> Info
	remote   *remote.OCIRemote
	rootHash Digest
	cacheDir string
	dirty    atomic.Bool
}

// Open creates or opens a store for the given image ref (e.g., "ttl.sh/cache/go:main").
func Open(imageRef string, opts ...OpenOption) (FS, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	cacheDir := expandPath(options.CacheDir)
	blobDir := filepath.Join(cacheDir, "blobs", "sha256")
	if err := os.MkdirAll(blobDir, 0755); err != nil {
		return nil, fmt.Errorf("create blob dir: %w", err)
	}

	auth := options.Auth
	if auth == nil {
		auth = remote.NewDefaultAuthenticator()
	}

	ociRemote, err := remote.NewOCIRemote(imageRef, auth)
	if err != nil {
		return nil, err
	}
	ociRemote.SetConcurrency(options.Concurrency)

	s := &CAS{
		blobs:    &blobStore{dir: blobDir},
		remote:   ociRemote,
		cacheDir: cacheDir,
	}

	if err := s.loadLocalIndex(); err == nil {
		s.rootHash = s.Hash("")
	}

	if options.AutoPull == AutoPullAlways || options.AutoPull == AutoPullMissing {
		_ = s.Pull(context.Background())
	}

	return s, nil
}

// Put stores data at key with optional metadata.
func (s *CAS) Put(key string, data []byte, opts ...Option) error {
	digest, err := s.blobs.Put(data)
	if err != nil {
		return err
	}

	info := Info{
		Digest: digest,
		Size:   int64(len(data)),
	}

	for _, opt := range opts {
		opt(&info)
	}

	s.entries.Store(key, info)
	s.dirty.Store(true)
	return nil
}

// Get retrieves data by key.
func (s *CAS) Get(key string) ([]byte, error) {
	v, ok := s.entries.Load(key)
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	info := v.(Info)
	return s.blobs.Get(info.Digest)
}

// Stat returns metadata for key.
func (s *CAS) Stat(key string) (Info, bool) {
	v, ok := s.entries.Load(key)
	if !ok {
		return Info{}, false
	}
	return v.(Info), true
}

// Delete removes an entry by key.
func (s *CAS) Delete(key string) {
	s.entries.Delete(key)
	s.dirty.Store(true)
}

// List iterates entries matching prefix.
func (s *CAS) List(prefix string) iter.Seq2[string, Info] {
	return func(yield func(string, Info) bool) {
		s.entries.Range(func(k, v any) bool {
			key := k.(string)
			if strings.HasPrefix(key, prefixHashKeyPrefix) {
				return true // skip internal prefix hashes
			}
			if rel, ok := strings.CutPrefix(key, prefix); ok {
				return yield(rel, v.(Info))
			}
			return true
		})
	}
}

// Hash computes merkle hash for prefix.
func (s *CAS) Hash(prefix string) Digest {
	var items []string
	s.entries.Range(func(k, v any) bool {
		key := k.(string)
		if strings.HasPrefix(key, prefixHashKeyPrefix) {
			return true // skip internal prefix hashes
		}
		if rel, ok := strings.CutPrefix(key, prefix); ok {
			info := v.(Info)
			// Include digest + size in hash (metadata is optional/variable)
			items = append(items, fmt.Sprintf("%s\x00%s\x00%d", rel, info.Digest, info.Size))
		}
		return true
	})
	if len(items) == 0 {
		return ""
	}
	sort.Strings(items)
	content := strings.Join(items, "\n")
	h := sha256.Sum256([]byte(content))
	return Digest(digestPrefix + hex.EncodeToString(h[:]))
}

func (s *CAS) Root() Digest { return s.Hash("") }
func (s *CAS) Dirty() bool  { return s.dirty.Load() }
func (s *CAS) Close() error { return s.Sync() }

// Path returns the filesystem path for a digest (for advanced use cases).
func (s *CAS) Path(digest Digest) string {
	return s.blobs.blobPath(digest)
}

func (s *CAS) Sync() error {
	if !s.dirty.Load() {
		return nil
	}

	indexPath := s.indexPath()
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return fmt.Errorf("create index dir: %w", err)
	}

	data, err := s.serialize()
	if err != nil {
		return fmt.Errorf("serialize index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	s.dirty.Store(false)
	return nil
}

func (s *CAS) indexPath() string {
	name := strings.ReplaceAll(s.remote.String(), "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return filepath.Join(s.cacheDir, "index", name+".json")
}

// Push uploads to the specified tags.
func (s *CAS) Push(ctx context.Context, tags ...string) error {
	if s.remote == nil {
		return ErrNoRemote
	}
	if len(tags) == 0 {
		tags = []string{s.remote.Tag()}
	}
	for _, tag := range tags {
		if err := s.pushToTag(ctx, tag); err != nil {
			return err
		}
	}
	return nil
}

func (s *CAS) pushToTag(ctx context.Context, tag string) error {
	indexData, err := s.serialize()
	if err != nil {
		return fmt.Errorf("serialize index: %w", err)
	}

	indexDigest, err := s.blobs.Put(indexData)
	if err != nil {
		return fmt.Errorf("store index: %w", err)
	}

	objects := map[string][]byte{string(indexDigest): indexData}
	s.blobs.pending.Range(func(k, _ any) bool {
		digest := k.(Digest)
		if data, err := s.blobs.Get(digest); err == nil {
			objects[string(digest)] = data
		}
		return true
	})

	r, err := s.remote.WithTag(tag)
	if err != nil {
		return fmt.Errorf("invalid tag %q: %w", tag, err)
	}

	newPrefixes, err := r.Push(ctx, string(indexDigest), objects, s.loadPrefixHashes())
	if err != nil {
		return fmt.Errorf("push to %s: %w", tag, err)
	}

	s.savePrefixHashes(newPrefixes)
	s.blobs.pending = sync.Map{}
	s.rootHash = indexDigest
	return nil
}

// Pull downloads from remote.
func (s *CAS) Pull(ctx context.Context) error {
	if s.remote == nil {
		return ErrNoRemote
	}

	indexHash, objects, newPrefixes, err := s.remote.Pull(ctx, s.loadPrefixHashes())
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	for hash, data := range objects {
		if _, err := s.blobs.putWithDigest(normalizeDigest(hash), data); err != nil {
			return fmt.Errorf("store blob %s: %w", hash, err)
		}
	}

	s.savePrefixHashes(newPrefixes)

	indexDigest := normalizeDigest(indexHash)
	indexData, ok := objects[indexHash]
	if !ok {
		var err error
		indexData, err = s.blobs.Get(indexDigest)
		if err != nil {
			return fmt.Errorf("load index: %w", err)
		}
	}

	if err := s.load(indexData); err != nil {
		return fmt.Errorf("parse index: %w", err)
	}

	s.dirty.Store(true)
	if err := s.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	s.rootHash = indexDigest
	return nil
}

func (s *CAS) loadLocalIndex() error {
	data, err := os.ReadFile(s.indexPath())
	if err != nil {
		return err
	}
	return s.load(data)
}

const prefixHashKeyPrefix = "_prefix/"

func (s *CAS) loadPrefixHashes() map[string]remote.PrefixInfo {
	result := make(map[string]remote.PrefixInfo)
	s.entries.Range(func(k, v any) bool {
		key := k.(string)
		if strings.HasPrefix(key, prefixHashKeyPrefix) {
			prefix := strings.TrimPrefix(key, prefixHashKeyPrefix)
			info := v.(Info)
			parts := strings.SplitN(string(info.Digest), "|", 2)
			if len(parts) == 2 {
				result[prefix] = remote.PrefixInfo{Hash: parts[0], Layer: parts[1]}
			}
		}
		return true
	})
	return result
}

func (s *CAS) savePrefixHashes(prefixes map[string]remote.PrefixInfo) {
	for prefix, info := range prefixes {
		key := prefixHashKeyPrefix + prefix
		s.entries.Store(key, Info{Digest: Digest(info.Hash + "|" + info.Layer)})
	}
	s.dirty.Store(true)
}

// Serialization format
type serializedInfo struct {
	Digest string `json:"d"`
	Size   int64  `json:"s,omitempty"`
	Meta   any    `json:"m,omitempty"`
}

func (s *CAS) serialize() ([]byte, error) {
	m := make(map[string]serializedInfo)
	s.entries.Range(func(k, v any) bool {
		info := v.(Info)
		m[k.(string)] = serializedInfo{
			Digest: string(info.Digest),
			Size:   info.Size,
			Meta:   info.Meta,
		}
		return true
	})
	return json.Marshal(m)
}

func (s *CAS) load(data []byte) error {
	var m map[string]serializedInfo
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range m {
		s.entries.Store(k, Info{
			Digest: Digest(v.Digest),
			Size:   v.Size,
			Meta:   v.Meta,
		})
	}
	return nil
}

// blobStore handles content-addressed blob storage
type blobStore struct {
	dir     string
	pending sync.Map
}

func (b *blobStore) Put(data []byte) (Digest, error) {
	h := sha256.Sum256(data)
	digest := Digest(digestPrefix + hex.EncodeToString(h[:]))
	isNew, err := b.putWithDigest(digest, data)
	if err != nil {
		return "", err
	}
	if isNew {
		b.pending.Store(digest, struct{}{})
	}
	return digest, nil
}

func (b *blobStore) putWithDigest(digest Digest, data []byte) (isNew bool, err error) {
	path := b.blobPath(digest)
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, data, 0644)
}

func (b *blobStore) Get(digest Digest) ([]byte, error) {
	return os.ReadFile(b.blobPath(digest))
}

func (b *blobStore) blobPath(digest Digest) string {
	hash := strings.TrimPrefix(string(digest), digestPrefix)
	if len(hash) < 4 {
		return filepath.Join(b.dir, hash)
	}
	return filepath.Join(b.dir, hash[:2], hash[2:])
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func normalizeDigest(hash string) Digest {
	if strings.HasPrefix(hash, digestPrefix) {
		return Digest(hash)
	}
	return Digest(digestPrefix + hash)
}
