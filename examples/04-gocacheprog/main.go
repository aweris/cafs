package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aweris/cafs"
)

type Cmd string

const (
	CmdGet   Cmd = "get"
	CmdPut   Cmd = "put"
	CmdClose Cmd = "close"
)

type Request struct {
	ID       int64  `json:"ID"`
	Command  Cmd    `json:"Command"`
	ActionID []byte `json:"ActionID,omitempty"`
	OutputID []byte `json:"OutputID,omitempty"`
	BodySize int64  `json:"BodySize,omitempty"`
}

type Response struct {
	ID            int64  `json:"ID"`
	Miss          bool   `json:"Miss,omitempty"`
	OutputID      []byte `json:"OutputID,omitempty"`
	DiskPath      string `json:"DiskPath,omitempty"`
	Size          int64  `json:"Size,omitempty"`
	Err           string `json:"Err,omitempty"`
	KnownCommands []Cmd  `json:"KnownCommands,omitempty"`
}

const metaPrefix = "meta:"

func main() {
	// Full image ref, e.g., "ttl.sh/gocache/default:main" or "gocache/default:main" for local
	imageRef := envOr("GOCACHEPROG_REF", "gocache/default:main")
	cacheDir := envOr("GOCACHEPROG_DIR", defaultCacheDir())
	pull := envBool("GOCACHEPROG_PULL")
	push := envBool("GOCACHEPROG_PUSH")

	opts := []cafs.Option{cafs.WithCacheDir(cacheDir)}
	if pull {
		opts = append(opts, cafs.WithAutoPull("always"))
	}

	fs, err := cafs.Open(imageRef, opts...)
	if err != nil {
		fatal(err)
	}
	defer fs.Close()

	if push {
		defer func() {
			fs.Push(context.Background()) // pushes to current tag
		}()
	}

	if err := run(os.Stdin, os.Stdout, fs); err != nil {
		fatal(err)
	}
}

func run(r io.Reader, w io.Writer, fs cafs.FS) error {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)

	enc.Encode(Response{KnownCommands: []Cmd{CmdGet, CmdPut, CmdClose}})

	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var body []byte
		if req.BodySize > 0 {
			var bodyBase64 string
			if err := dec.Decode(&bodyBase64); err != nil {
				return err
			}
			var err error
			body, err = base64.StdEncoding.DecodeString(bodyBase64)
			if err != nil {
				return err
			}
		}

		resp := handle(req, body, fs)
		if err := enc.Encode(resp); err != nil {
			return err
		}

		if req.Command == CmdClose {
			return nil
		}
	}
}

func handle(req Request, body []byte, fs cafs.FS) Response {
	switch req.Command {
	case CmdGet:
		return handleGet(req, fs)
	case CmdPut:
		return handlePut(req, body, fs)
	case CmdClose:
		return Response{ID: req.ID}
	default:
		return Response{ID: req.ID, Err: "unknown command"}
	}
}

func handleGet(req Request, fs cafs.FS) Response {
	actionID := hex.EncodeToString(req.ActionID)

	meta, ok := fs.Index().Get(actionID)
	if !ok {
		return Response{ID: req.ID, Miss: true}
	}

	// Parse meta:outputID:bodyDigest:size
	metaStr := string(meta)
	if !strings.HasPrefix(metaStr, metaPrefix) {
		return Response{ID: req.ID, Miss: true}
	}
	parts := strings.SplitN(metaStr[len(metaPrefix):], ":", 3)
	if len(parts) != 3 {
		return Response{ID: req.ID, Miss: true}
	}

	outputIDHex, bodyDigestStr := parts[0], parts[1]+":"+parts[2][:64]
	bodyDigest := cafs.Digest(bodyDigestStr)

	size, exists := fs.Blobs().Stat(bodyDigest)
	if !exists {
		return Response{ID: req.ID, Miss: true}
	}

	outputID, _ := hex.DecodeString(outputIDHex)
	return Response{
		ID:       req.ID,
		OutputID: outputID,
		DiskPath: fs.Blobs().Path(bodyDigest),
		Size:     size,
	}
}

func handlePut(req Request, body []byte, fs cafs.FS) Response {
	actionID := hex.EncodeToString(req.ActionID)
	outputID := hex.EncodeToString(req.OutputID)

	var bodyDigest cafs.Digest
	var err error

	if len(body) > 0 {
		bodyDigest, err = fs.Blobs().Put(body)
		if err != nil {
			return Response{ID: req.ID, Err: err.Error()}
		}
	} else {
		bodyDigest = cafs.Digest("sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		fs.Blobs().Put([]byte{})
	}

	// Store meta directly in index: meta:outputID:bodyDigest
	meta := cafs.Digest(fmt.Sprintf("%s%s:%s", metaPrefix, outputID, bodyDigest))
	fs.Index().Set(actionID, meta)

	return Response{
		ID:       req.ID,
		DiskPath: fs.Blobs().Path(bodyDigest),
		Size:     int64(len(body)),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string) bool {
	v := os.Getenv(key)
	return v == "1" || v == "true"
}

func defaultCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "cafs")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "cafs")
	}
	return ".cafs"
}

func fatal(err error) {
	os.Stderr.WriteString("gocacheprog: " + err.Error() + "\n")
	os.Exit(1)
}
