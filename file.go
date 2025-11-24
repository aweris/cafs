package cafs

import (
	"io"
	"io/fs"
)

type file struct {
	node   *Node
	ws     *Workspace
	offset int64
	flag   int
}

func newFile(node *Node, ws *Workspace, flag int) *file {
	return &file{
		node:   node,
		ws:     ws,
		offset: 0,
		flag:   flag,
	}
}

func (f *file) Read(p []byte) (n int, err error) {
	if f.node.IsDir() {
		return 0, fs.ErrInvalid
	}

	content := f.node.Content()
	if f.offset >= int64(len(content)) {
		return 0, io.EOF
	}

	n = copy(p, content[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *file) Write(p []byte) (n int, err error) {
	// TODO: Implement
	// 1. Append/overwrite to node content
	// 2. Mark node and path as dirty
	return 0, ErrNotImplemented
}

func (f *file) Close() error {
	return nil
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	// TODO: Implement
	return 0, ErrNotImplemented
}

func (f *file) ReadAt(p []byte, off int64) (n int, err error) {
	// TODO: Implement
	return 0, ErrNotImplemented
}

func (f *file) Name() string {
	return f.node.Name()
}

func (f *file) Stat() (fs.FileInfo, error) {
	return f.node, nil
}

// No-op for in-memory.
func (f *file) Sync() error {
	return nil
}

func (f *file) Truncate(size int64) error {
	// TODO: Implement
	return ErrNotImplemented
}

func (f *file) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}

func (f *file) Readdir(count int) ([]fs.FileInfo, error) {
	// TODO: Implement
	return nil, ErrNotImplemented
}

func (f *file) Readdirnames(n int) ([]string, error) {
	// TODO: Implement
	return nil, ErrNotImplemented
}

func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	// TODO: Implement
	return nil, ErrNotImplemented
}

func (f *file) Info() (fs.FileInfo, error) {
	return f.node, nil
}

// Node implements fs.FileInfo interface
var _ fs.FileInfo = (*Node)(nil)

func (n *Node) Info() (fs.FileInfo, error) {
	return n, nil
}

func (n *Node) Type() fs.FileMode {
	return n.mode.Type()
}

func (n *Node) Sys() any {
	return nil
}
