package remote

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type OCIRemote struct {
	registry  string // e.g., "registry.io"
	namespace string // e.g., "myorg/project"
	ref       string // e.g., "main"
	auth      Authenticator
}

func NewOCIRemote(registry, namespace, ref string, auth Authenticator) *OCIRemote {
	return &OCIRemote{
		registry:  registry,
		namespace: namespace,
		ref:       ref,
		auth:      auth,
	}
}

// blobLayer implements v1.Layer for raw blob content.
type blobLayer struct {
	content   []byte
	mediaType types.MediaType
}

func (l *blobLayer) Digest() (v1.Hash, error) {
	hash, _, err := v1.SHA256(bytes.NewReader(l.content))
	return hash, err
}

func (l *blobLayer) DiffID() (v1.Hash, error) {
	return l.Digest()
}

func (l *blobLayer) Compressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.content)), nil
}

func (l *blobLayer) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.content)), nil
}

func (l *blobLayer) Size() (int64, error) {
	return int64(len(l.content)), nil
}

func (l *blobLayer) MediaType() (types.MediaType, error) {
	return l.mediaType, nil
}

func (r *OCIRemote) Push(ctx context.Context, rootHash string, objects map[string][]byte) error {
	repoName := fmt.Sprintf("%s/%s", r.registry, r.namespace)

	layers := make([]v1.Layer, 0, len(objects))
	for _, data := range objects {
		layer := &blobLayer{
			content:   data,
			mediaType: types.OCILayer,
		}
		layers = append(layers, layer)
	}

	img, err := r.buildImage(layers, rootHash)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	tag, err := name.NewTag(fmt.Sprintf("%s:%s", repoName, r.ref))
	if err != nil {
		return fmt.Errorf("invalid tag: %w", err)
	}

	options := []remote.Option{}
	if r.auth != nil {
		username, password, err := r.auth.Authenticate(r.registry)
		if err == nil && username != "" {
			options = append(options, remote.WithAuth(&authn.Basic{
				Username: username,
				Password: password,
			}))
		}
	}

	if err := remote.Write(tag, img, options...); err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}

	return nil
}

func (r *OCIRemote) buildImage(layers []v1.Layer, rootHash string) (v1.Image, error) {
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

	cfg.Config.Labels = map[string]string{
		"dev.cafs.root.hash": rootHash,
	}

	img, err = mutate.ConfigFile(img, cfg)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (r *OCIRemote) Pull(ctx context.Context) (string, map[string][]byte, error) {
	repoName := fmt.Sprintf("%s/%s", r.registry, r.namespace)
	tag, err := name.NewTag(fmt.Sprintf("%s:%s", repoName, r.ref))
	if err != nil {
		return "", nil, fmt.Errorf("invalid tag: %w", err)
	}

	options := []remote.Option{}
	if r.auth != nil {
		username, password, err := r.auth.Authenticate(r.registry)
		if err == nil && username != "" {
			options = append(options, remote.WithAuth(&authn.Basic{
				Username: username,
				Password: password,
			}))
		}
	}

	img, err := remote.Image(tag, options...)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch image: %w", err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get config: %w", err)
	}

	rootHash, ok := cfg.Config.Labels["dev.cafs.root.hash"]
	if !ok {
		return "", nil, fmt.Errorf("missing dev.cafs.root.hash label")
	}

	layers, err := img.Layers()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get layers: %w", err)
	}

	objects := make(map[string][]byte)
	for _, layer := range layers {
		rc, err := layer.Uncompressed()
		if err != nil {
			return "", nil, fmt.Errorf("failed to read layer: %w", err)
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", nil, fmt.Errorf("failed to read layer data: %w", err)
		}

		hash := sha256.Sum256(data)
		hashStr := hex.EncodeToString(hash[:])

		objects[hashStr] = data
	}

	return rootHash, objects, nil
}

// Not implemented - Pull() fetches by tag directly.
func (r *OCIRemote) GetRef(ctx context.Context) (string, error) {
	return "", fmt.Errorf("not implemented")
}
