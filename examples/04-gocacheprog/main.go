package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

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

type cacheMeta struct {
	OutputID string `json:"o" mapstructure:"o"`
}

func main() {
	imageRef := envOr("GOCACHEPROG_REF", "gocache/default:main")
	cacheDir := envOr("GOCACHEPROG_DIR", defaultCacheDir())

	fs, err := cafs.Open(imageRef, cafs.WithCacheDir(cacheDir))
	if err != nil {
		fatal(err)
	}
	defer fs.Close()

	if err := run(os.Stdin, os.Stdout, fs); err != nil {
		fatal(err)
	}
}

func run(r io.Reader, w io.Writer, fs cafs.Store) error {
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

func handle(req Request, body []byte, fs cafs.Store) Response {
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

func handleGet(req Request, fs cafs.Store) Response {
	actionID := hex.EncodeToString(req.ActionID)

	info, ok := fs.Stat(actionID)
	if !ok {
		return Response{ID: req.ID, Miss: true}
	}

	var meta cacheMeta
	if err := info.DecodeMeta(&meta); err != nil {
		return Response{ID: req.ID, Miss: true}
	}

	outputID, err := hex.DecodeString(meta.OutputID)
	if err != nil {
		return Response{ID: req.ID, Miss: true}
	}

	return Response{
		ID:       req.ID,
		OutputID: outputID,
		DiskPath: fs.Path(info.Digest),
		Size:     info.Size,
	}
}

func handlePut(req Request, body []byte, fs cafs.Store) Response {
	actionID := hex.EncodeToString(req.ActionID)
	outputID := hex.EncodeToString(req.OutputID)

	meta := cacheMeta{OutputID: outputID}
	if err := fs.Put(actionID, body, cafs.WithMeta(meta)); err != nil {
		return Response{ID: req.ID, Err: err.Error()}
	}

	info, _ := fs.Stat(actionID)
	return Response{
		ID:       req.ID,
		DiskPath: fs.Path(info.Digest),
		Size:     info.Size,
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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
