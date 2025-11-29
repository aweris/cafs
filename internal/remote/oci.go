package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/klauspost/compress/zstd"
	"github.com/sourcegraph/conc/pool"
)

const DefaultConcurrency = 4

type OCIRemote struct {
	ref         name.Reference
	auth        Authenticator
	concurrency int
}

// NewOCIRemote creates a remote from a standard Docker ref (e.g., "ttl.sh/cache/go:main")
func NewOCIRemote(imageRef string, auth Authenticator) (*OCIRemote, error) {
	ref, err := name.ParseReference(imageRef, name.WithDefaultTag("latest"))
	if err != nil {
		return nil, fmt.Errorf("invalid image ref %q: %w", imageRef, err)
	}
	return &OCIRemote{ref: ref, auth: auth, concurrency: DefaultConcurrency}, nil
}

// SetConcurrency sets the number of parallel operations for push/pull
func (r *OCIRemote) SetConcurrency(n int) {
	if n > 0 {
		r.concurrency = n
	}
}

func (r *OCIRemote) String() string   { return r.ref.String() }
func (r *OCIRemote) Registry() string { return r.ref.Context().RegistryStr() }
func (r *OCIRemote) Tag() string      { return r.ref.Identifier() }

// WithTag returns a new OCIRemote with a different tag
func (r *OCIRemote) WithTag(tag string) (*OCIRemote, error) {
	newRef, err := name.NewTag(r.ref.Context().String()+":"+tag, name.WithDefaultTag("latest"))
	if err != nil {
		return nil, err
	}
	return &OCIRemote{ref: newRef, auth: r.auth, concurrency: r.concurrency}, nil
}

// blobLayer implements v1.Layer with zstd compression for remote transfer
type blobLayer struct {
	compressed   []byte
	uncompressed []byte
}

var zstdEncoder, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))

func newBlobLayer(data []byte) *blobLayer {
	return &blobLayer{
		compressed:   zstdEncoder.EncodeAll(data, nil),
		uncompressed: data,
	}
}

func (l *blobLayer) Digest() (v1.Hash, error) {
	h, _, err := v1.SHA256(bytes.NewReader(l.compressed))
	return h, err
}

func (l *blobLayer) DiffID() (v1.Hash, error) {
	h, _, err := v1.SHA256(bytes.NewReader(l.uncompressed))
	return h, err
}

func (l *blobLayer) Compressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.compressed)), nil
}
func (l *blobLayer) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.uncompressed)), nil
}
func (l *blobLayer) Size() (int64, error)                { return int64(len(l.compressed)), nil }
func (l *blobLayer) MediaType() (types.MediaType, error) { return types.OCILayerZStd, nil }

// Push uploads blobs incrementally based on prefix hashes
func (r *OCIRemote) Push(ctx context.Context, rootHash string, objects map[string][]byte, localPrefixes map[string]PrefixInfo) (map[string]PrefixInfo, error) {
	// Group blobs by prefix
	byPrefix := GroupByPrefix(objects)

	fmt.Fprintf(os.Stderr, "[push] %d blobs across %d prefixes\n", len(objects), len(byPrefix))

	// Compute current prefix hashes
	currentHashes := make(map[string]string)
	for prefix, blobs := range byPrefix {
		currentHashes[prefix] = PrefixHash(blobs)
	}

	// Find changed prefixes
	var changedPrefixes []string
	for prefix, hash := range currentHashes {
		if local, ok := localPrefixes[prefix]; !ok || local.Hash != hash {
			changedPrefixes = append(changedPrefixes, prefix)
		}
	}

	fmt.Fprintf(os.Stderr, "[push] %d prefixes changed (of %d local)\n", len(changedPrefixes), len(localPrefixes))

	// Build result with existing layer refs for unchanged prefixes
	newPrefixes := make(map[string]PrefixInfo)
	for prefix, info := range localPrefixes {
		if _, exists := currentHashes[prefix]; !exists {
			continue // prefix no longer exists
		}
		newPrefixes[prefix] = info
	}

	// If nothing changed, just update manifest
	if len(changedPrefixes) == 0 {
		fmt.Fprintf(os.Stderr, "[push] no changes, updating manifest only\n")
		return newPrefixes, r.pushManifest(ctx, rootHash, newPrefixes)
	}

	// Collect blobs from changed prefixes
	changedByPrefix := make(map[string]map[string][]byte)
	for _, prefix := range changedPrefixes {
		changedByPrefix[prefix] = byPrefix[prefix]
	}

	// Build layer plan for changed prefixes
	sizes := CalculatePrefixSizes(changedByPrefix)
	layerPlan := BuildLayerPlan(sizes)

	fmt.Fprintf(os.Stderr, "[push] packing into %d layers\n", len(layerPlan))

	// Create layers
	layers := make([]v1.Layer, 0, len(layerPlan))
	var totalRaw, totalCompressed int64
	for _, prefixGroup := range layerPlan {
		blobs := CollectPrefixBlobs(prefixGroup, changedByPrefix)
		layerData := PackLayer(blobs)
		layer := newBlobLayer(layerData)
		digest, _ := layer.Digest()
		totalRaw += int64(len(layerData))
		totalCompressed += int64(len(layer.compressed))

		layers = append(layers, layer)
		for _, prefix := range prefixGroup {
			newPrefixes[prefix] = PrefixInfo{
				Hash:  currentHashes[prefix],
				Layer: digest.String(),
			}
		}
	}

	ratio := float64(totalCompressed) / float64(totalRaw) * 100
	fmt.Fprintf(os.Stderr, "[push] uploading %d layers (%.1fMB → %.1fMB, %.0f%%)\n",
		len(layers), float64(totalRaw)/(1024*1024), float64(totalCompressed)/(1024*1024), ratio)

	// Build and push image
	img, err := r.buildImage(layers, rootHash, newPrefixes)
	if err != nil {
		return nil, fmt.Errorf("build image: %w", err)
	}

	if err := r.pushImage(ctx, img); err != nil {
		return nil, fmt.Errorf("push image: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[push] done\n")
	return newPrefixes, nil
}

// pushManifest pushes just the manifest without new layers
func (r *OCIRemote) pushManifest(ctx context.Context, rootHash string, prefixes map[string]PrefixInfo) error {
	img, err := r.buildImage(nil, rootHash, prefixes)
	if err != nil {
		return err
	}
	return r.pushImage(ctx, img)
}

func (r *OCIRemote) buildImage(layers []v1.Layer, rootHash string, prefixes map[string]PrefixInfo) (v1.Image, error) {
	img := empty.Image

	if len(layers) > 0 {
		var err error
		img, err = mutate.AppendLayers(img, layers...)
		if err != nil {
			return nil, err
		}
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	prefixJSON, _ := json.Marshal(prefixes)

	cfg.Config.Labels = map[string]string{
		"dev.cafs.root":     rootHash,
		"dev.cafs.prefixes": string(prefixJSON),
	}

	return mutate.ConfigFile(img, cfg)
}

func (r *OCIRemote) pushImage(ctx context.Context, img v1.Image) error {
	options := r.remoteOptions()
	options = append(options, remote.WithJobs(r.concurrency))
	_, err := retry(ctx, 3, func() (struct{}, error) {
		return struct{}{}, remote.Write(r.ref, img, options...)
	})
	return err
}

// Pull downloads blobs incrementally based on prefix hashes
func (r *OCIRemote) Pull(ctx context.Context, localPrefixes map[string]PrefixInfo) (string, map[string][]byte, map[string]PrefixInfo, error) {
	img, err := retry(ctx, 3, func() (v1.Image, error) {
		return remote.Image(r.ref, r.remoteOptions()...)
	})
	if err != nil {
		return "", nil, nil, fmt.Errorf("fetch image: %w", err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return "", nil, nil, fmt.Errorf("get config: %w", err)
	}

	rootHash := cfg.Config.Labels["dev.cafs.root"]
	if rootHash == "" {
		return "", nil, nil, fmt.Errorf("missing dev.cafs.root label")
	}

	var remotePrefixes map[string]PrefixInfo
	if prefixJSON := cfg.Config.Labels["dev.cafs.prefixes"]; prefixJSON != "" {
		if err := json.Unmarshal([]byte(prefixJSON), &remotePrefixes); err != nil {
			return "", nil, nil, fmt.Errorf("parse prefixes: %w", err)
		}
	}

	// Find layers we need to download
	neededLayers := make(map[string]bool)
	for prefix, remoteInfo := range remotePrefixes {
		localInfo, exists := localPrefixes[prefix]
		if !exists || localInfo.Hash != remoteInfo.Hash {
			neededLayers[remoteInfo.Layer] = true
		}
	}

	// Download needed layers in parallel
	layers, err := img.Layers()
	if err != nil {
		return "", nil, nil, fmt.Errorf("get layers: %w", err)
	}

	// Filter to needed layers
	var neededLayerList []v1.Layer
	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			continue
		}
		if neededLayers[digest.String()] {
			neededLayerList = append(neededLayerList, layer)
		}
	}

	fmt.Fprintf(os.Stderr, "[pull] downloading %d layers in parallel\n", len(neededLayerList))

	// Download in parallel using conc pool
	var mu sync.Mutex
	objects := make(map[string][]byte)

	p := pool.New().WithMaxGoroutines(r.concurrency).WithContext(ctx).WithCancelOnError()

	for _, layer := range neededLayerList {
		p.Go(func(ctx context.Context) error {
			rc, err := layer.Uncompressed()
			if err != nil {
				return fmt.Errorf("read layer: %w", err)
			}
			data, err := io.ReadAll(rc)
			if cerr := rc.Close(); cerr != nil {
				return fmt.Errorf("close layer: %w", cerr)
			}
			if err != nil {
				return fmt.Errorf("read layer: %w", err)
			}

			blobs, err := UnpackLayer(data)
			if err != nil {
				return fmt.Errorf("unpack layer: %w", err)
			}

			mu.Lock()
			for k, v := range blobs {
				objects[k] = v
			}
			mu.Unlock()
			return nil
		})
	}

	if err := p.Wait(); err != nil {
		return "", nil, nil, err
	}

	fmt.Fprintf(os.Stderr, "[pull] done, %d blobs received\n", len(objects))
	return rootHash, objects, remotePrefixes, nil
}

func (r *OCIRemote) remoteOptions() []remote.Option {
	if r.auth != nil {
		username, password, err := r.auth.Authenticate(r.Registry())
		if err == nil && username != "" {
			return []remote.Option{remote.WithAuth(&authn.Basic{
				Username: username,
				Password: password,
			})}
		}
	}
	return []remote.Option{remote.WithAuthFromKeychain(authn.DefaultKeychain)}
}

func retry[T any](ctx context.Context, maxAttempts int, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	for i := range maxAttempts {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err
		if i < maxAttempts-1 {
			delay := time.Duration(1<<i) * 500 * time.Millisecond // 500ms, 1s, 2s, 4s...
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return zero, lastErr
}
