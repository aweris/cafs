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
	idx      *index
	remote   *remote.OCIRemote
	rootHash Digest
	cacheDir string
}

// Open creates or opens a store for the given image ref (e.g., "ttl.sh/cache/go:main").
// Local-only usage: use a ref without registry like "cache/go:main".
func Open(imageRef string, opts ...Option) (FS, error) {
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
		idx:      &index{entries: sync.Map{}},
		remote:   ociRemote,
		cacheDir: cacheDir,
	}

	if err := s.loadLocalIndex(); err == nil {
		s.rootHash = s.idx.Hash("")
	}

	if options.AutoPull == AutoPullAlways || options.AutoPull == AutoPullMissing {
		_ = s.Pull(context.Background())
	}

	return s, nil
}

func (s *CAS) Blobs() BlobStore { return s.blobs }
func (s *CAS) Index() Index     { return s.idx }
func (s *CAS) Root() Digest     { return s.idx.Hash("") }
func (s *CAS) Dirty() bool      { return s.idx.dirty.Load() }

func (s *CAS) Sync() error {
	if !s.idx.dirty.Load() {
		return nil
	}

	indexPath := s.indexPath()
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return fmt.Errorf("create index dir: %w", err)
	}

	data, err := s.idx.serialize()
	if err != nil {
		return fmt.Errorf("serialize index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	s.idx.dirty.Store(false)
	return nil
}

func (s *CAS) indexPath() string {
	// Use ref string as filename (sanitized: replace / and : with _)
	name := strings.ReplaceAll(s.remote.String(), "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	return filepath.Join(s.cacheDir, "index", name+".json")
}

// Push uploads to the specified tags. If no tags provided, uses the current ref's tag.
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

func (s *CAS) Close() error {
	return s.Sync()
}

func (s *CAS) pushToTag(ctx context.Context, tag string) error {
	indexData, err := s.idx.serialize()
	if err != nil {
		return fmt.Errorf("serialize index: %w", err)
	}

	indexDigest, err := s.blobs.Put(indexData)
	if err != nil {
		return fmt.Errorf("store index: %w", err)
	}

	// Collect all blobs (index + pending)
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

func (s *CAS) Pull(ctx context.Context) error {
	if s.remote == nil {
		return ErrNoRemote
	}

	indexHash, objects, newPrefixes, err := s.remote.Pull(ctx, s.loadPrefixHashes())
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	// Store received blobs
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

	if err := s.idx.load(indexData); err != nil {
		return fmt.Errorf("parse index: %w", err)
	}

	s.idx.dirty.Store(true)
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
	return s.idx.load(data)
}

const prefixHashKeyPrefix = "_prefix/"

func (s *CAS) loadPrefixHashes() map[string]remote.PrefixInfo {
	result := make(map[string]remote.PrefixInfo)
	s.idx.entries.Range(func(k, v any) bool {
		key := k.(string)
		if strings.HasPrefix(key, prefixHashKeyPrefix) {
			prefix := strings.TrimPrefix(key, prefixHashKeyPrefix)
			// Value format: "hash:layer"
			parts := strings.SplitN(string(v.(Digest)), "|", 2)
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
		value := Digest(info.Hash + "|" + info.Layer)
		s.idx.entries.Store(key, value)
	}
	s.idx.dirty.Store(true)
}

// blobStore implements BlobStore
type blobStore struct {
	dir     string
	pending sync.Map // tracks digests since last push
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
	path := b.Path(digest)
	if _, err := os.Stat(path); err == nil {
		return false, nil // already exists
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, data, 0644)
}

func (b *blobStore) Get(digest Digest) ([]byte, error) {
	return os.ReadFile(b.Path(digest))
}

func (b *blobStore) Stat(digest Digest) (int64, bool) {
	info, err := os.Stat(b.Path(digest))
	if err != nil {
		return 0, false
	}
	return info.Size(), true
}

func (b *blobStore) Path(digest Digest) string {
	hash := strings.TrimPrefix(string(digest), digestPrefix)
	if len(hash) < 4 {
		return filepath.Join(b.dir, hash)
	}
	return filepath.Join(b.dir, hash[:2], hash[2:])
}

// index implements Index
type index struct {
	entries sync.Map
	dirty   atomic.Bool
}

func (i *index) Set(key string, digest Digest) {
	i.entries.Store(key, digest)
	i.dirty.Store(true)
}

func (i *index) Get(key string) (Digest, bool) {
	v, ok := i.entries.Load(key)
	if !ok {
		return "", false
	}
	return v.(Digest), true
}

func (i *index) Delete(key string) {
	i.entries.Delete(key)
	i.dirty.Store(true)
}

func (i *index) Entries() iter.Seq2[string, Digest] {
	return func(yield func(string, Digest) bool) {
		i.entries.Range(func(k, v any) bool {
			return yield(k.(string), v.(Digest))
		})
	}
}

func (i *index) Hash(prefix string) Digest {
	var items []string
	i.entries.Range(func(k, v any) bool {
		key := k.(string)
		if rel, ok := strings.CutPrefix(key, prefix); ok {
			items = append(items, rel+"\x00"+string(v.(Digest)))
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

func (i *index) List(prefix string) iter.Seq2[string, Digest] {
	return func(yield func(string, Digest) bool) {
		i.entries.Range(func(k, v any) bool {
			key := k.(string)
			if rel, ok := strings.CutPrefix(key, prefix); ok {
				return yield(rel, v.(Digest))
			}
			return true
		})
	}
}

func (i *index) serialize() ([]byte, error) {
	m := make(map[string]string)
	i.entries.Range(func(k, v any) bool {
		m[k.(string)] = string(v.(Digest))
		return true
	})
	return json.Marshal(m)
}

func (i *index) load(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range m {
		i.entries.Store(k, Digest(v))
	}
	return nil
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
