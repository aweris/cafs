package cafs

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Workspace is the main implementation of the FS interface.
// It provides a mutable view of the filesystem that feels like working
// with a local directory, while maintaining all changes in memory until
// explicitly pushed to remote storage.
type Workspace struct {
	namespace string
	ref       string

	store  Store
	remote Remote

	root *Node

	dirty    map[string]struct{}
	baseHash string

	mu sync.RWMutex
}

func newWorkspace(namespace, ref string, store Store, remote Remote) *Workspace {
	return &Workspace{
		namespace: namespace,
		ref:       ref,
		store:     store,
		remote:    remote,
		root:      newDirNode("", 0755),
		dirty:     make(map[string]struct{}),
	}
}

func (w *Workspace) Open(name string) (fs.File, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	node, err := w.navigate(cleanPath(name))
	if err != nil {
		return nil, err
	}

	if !node.loaded && node.hash != "" {
		if err := node.ensureLoaded(); err != nil {
			return nil, err
		}
	}

	return newFile(node, w, 0), nil
}

func (w *Workspace) Stat(name string) (fs.FileInfo, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	node, err := w.navigate(cleanPath(name))
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (w *Workspace) ReadDir(name string) ([]fs.DirEntry, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	node, err := w.navigate(cleanPath(name))
	if err != nil {
		return nil, err
	}

	if !node.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", name)
	}

	if err := node.ensureLoaded(); err != nil {
		return nil, fmt.Errorf("failed to load directory: %w", err)
	}

	children := node.Children()
	entries := make([]fs.DirEntry, 0, len(children))
	for _, child := range children {
		entries = append(entries, child)
	}

	return entries, nil
}

func (w *Workspace) ReadFile(path string) ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	node, err := w.navigate(path)
	if err != nil {
		return nil, err
	}

	if node.IsDir() {
		return nil, fmt.Errorf("%s: is a directory", path)
	}

	if err := node.ensureLoaded(); err != nil {
		return nil, fmt.Errorf("failed to load file: %w", err)
	}

	return node.Content(), nil
}

func (w *Workspace) Create(name string) (fs.File, error) {
	// TODO: Implement
	return nil, fs.ErrPermission
}

func (w *Workspace) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	// TODO: Implement
	return nil, fs.ErrNotExist
}

func (w *Workspace) WriteFile(name string, data []byte, perm fs.FileMode) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	name = cleanPath(name)
	dir, file := splitPath(name)

	parent, err := w.navigateOrCreate(dir, true)
	if err != nil {
		return err
	}

	if !parent.IsDir() {
		return fmt.Errorf("%s: not a directory", dir)
	}

	// Ensure parent directory's children are loaded (for lazy-loaded snapshots)
	if err := parent.ensureLoaded(); err != nil {
		return fmt.Errorf("failed to load directory %s: %w", dir, err)
	}

	if child, ok := parent.getChild(file); ok {
		if child.IsDir() {
			return fmt.Errorf("%s: is a directory", name)
		}

		if !child.loaded && child.hash != "" {
			if err := child.ensureLoaded(); err != nil {
				return fmt.Errorf("failed to load file %s: %w", name, err)
			}
		}

		contentChanged := !bytes.Equal(child.content, data)
		modeChanged := child.mode != perm

		if contentChanged || modeChanged {
			child.setContent(data)
			child.mode = perm
			w.markDirty(name)
		}
	} else {
		fileNode := newFileNode(file, data, perm)
		parent.addChild(file, fileNode)
		w.markDirty(name)
	}

	return nil
}

func (w *Workspace) Remove(name string) error {
	// TODO: Implement
	return fs.ErrPermission
}

func (w *Workspace) RemoveAll(path string) error {
	// TODO: Implement
	return fs.ErrPermission
}

func (w *Workspace) Rename(oldname, newname string) error {
	// TODO: Implement
	return fs.ErrPermission
}

func (w *Workspace) Mkdir(name string, perm fs.FileMode) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	name = cleanPath(name)
	dir, base := splitPath(name)

	parent, err := w.navigate(dir)
	if err != nil {
		return err
	}

	if !parent.IsDir() {
		return fmt.Errorf("%s: not a directory", dir)
	}

	if _, ok := parent.getChild(base); ok {
		return fmt.Errorf("%s: file exists", name)
	}

	dirNode := newDirNode(base, perm)
	parent.addChild(base, dirNode)
	w.markDirty(name)

	return nil
}

func (w *Workspace) MkdirAll(path string, perm fs.FileMode) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	existing, err := w.navigate(path)
	if err == nil && existing.IsDir() {
		return nil
	}

	_, err = w.navigateOrCreate(path, true)
	return err
}

func (w *Workspace) Chmod(name string, mode fs.FileMode) error {
	// TODO: Implement
	return fs.ErrPermission
}

func (w *Workspace) Chtimes(name string, atime time.Time, mtime time.Time) error {
	// TODO: Implement
	return fs.ErrPermission
}

// Sync operations

func (w *Workspace) collectObjects(ctx context.Context, node *Node, objects map[string][]byte) error {
	if node.hash == "" {
		return nil
	}

	if _, exists := objects[node.hash]; exists {
		return nil
	}

	data, err := w.store.Get(ctx, node.hash)
	if err != nil {
		return err
	}
	objects[node.hash] = data

	if node.IsDir() {
		for _, child := range node.Children() {
			if err := w.collectObjects(ctx, child, objects); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Workspace) Push(ctx context.Context) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// If nothing changed, return current hash
	if len(w.dirty) == 0 && w.baseHash != "" {
		return w.baseHash, nil
	}

	// Compute hashes bottom-up (also stores objects in local store)
	rootHash, err := w.root.computeHash(w.store)
	if err != nil {
		return "", fmt.Errorf("failed to compute tree hash: %w", err)
	}

	if err := w.store.PutRef(w.namespace, w.ref, rootHash); err != nil {
		return "", fmt.Errorf("failed to update ref: %w", err)
	}

	if w.remote != nil {
		objects := make(map[string][]byte)
		if err := w.collectObjects(ctx, w.root, objects); err != nil {
			return "", fmt.Errorf("failed to collect objects: %w", err)
		}

		if err := w.remote.Push(ctx, rootHash, objects); err != nil {
			return "", fmt.Errorf("failed to push to remote: %w", err)
		}
	}

	w.baseHash = rootHash
	w.dirty = make(map[string]struct{})

	return rootHash, nil
}

func (w *Workspace) Pull(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.dirty) > 0 {
		return fmt.Errorf("workspace has uncommitted changes, cannot pull")
	}

	if w.remote == nil {
		return fmt.Errorf("no remote configured")
	}

	rootHash, objects, err := w.remote.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull from remote: %w", err)
	}

	for hash, data := range objects {
		if _, err := w.store.Put(ctx, data); err != nil {
			return fmt.Errorf("failed to store object %s: %w", hash, err)
		}
	}

	if err := w.store.PutRef(w.namespace, w.ref, rootHash); err != nil {
		return fmt.Errorf("failed to update ref: %w", err)
	}

	snapshot := newSnapshot(rootHash, w.store)
	rootNode, err := snapshot.loadNode(rootHash)
	if err != nil {
		return fmt.Errorf("failed to load root node: %w", err)
	}

	w.root = rootNode
	w.baseHash = rootHash
	w.dirty = make(map[string]struct{})

	return nil
}

func (w *Workspace) Sync(ctx context.Context) (string, error) {
	// TODO: Implement (Pull then Push)
	return "", ErrNotImplemented
}

// Metadata queries

func (w *Workspace) IsDirty() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.dirty) > 0
}

func (w *Workspace) CurrentDigest() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.baseHash
}

func (w *Workspace) Namespace() string {
	return w.namespace
}

func (w *Workspace) Ref() string {
	return w.ref
}

// Internal helpers

func (w *Workspace) markDirty(path string) {
	w.dirty[path] = struct{}{}
}

func (w *Workspace) navigate(path string) (*Node, error) {
	path = cleanPath(path)
	if path == "/" {
		return w.root, nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := w.root

	for _, part := range parts {
		if part == "" {
			continue
		}

		if current.IsDir() {
			if err := current.ensureLoaded(); err != nil {
				return nil, fmt.Errorf("failed to load directory: %w", err)
			}
		}

		child, ok := current.getChild(part)
		if !ok {
			return nil, fs.ErrNotExist
		}

		current = child
	}

	return current, nil
}

func (w *Workspace) navigateOrCreate(path string, createDirs bool) (*Node, error) {
	path = cleanPath(path)
	if path == "/" {
		return w.root, nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := w.root

	for _, part := range parts {
		if part == "" {
			continue
		}

		child, ok := current.getChild(part)
		if !ok {
			if !createDirs {
				return nil, fs.ErrNotExist
			}
			child = newDirNode(part, 0755)
			current.addChild(part, child)
		}

		if !child.IsDir() {
			return nil, fmt.Errorf("%s: not a directory", part)
		}

		current = child
	}

	return current, nil
}

func cleanPath(path string) string {
	if path == "" || path == "." {
		return "/"
	}

	if path[0] == '/' && !strings.Contains(path, "..") && !strings.Contains(path, "//") {
		return path
	}

	path = filepath.Clean("/" + path)
	if path == "." {
		return "/"
	}
	return path
}

func splitPath(path string) (dir, file string) {
	path = cleanPath(path)
	if path == "/" {
		return "/", ""
	}
	dir, file = filepath.Split(path)
	if dir == "" {
		dir = "/"
	}
	return dir, file
}

func (w *Workspace) prefetchPath(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	node, err := w.navigate(cleanPath(path))
	if err != nil {
		return err
	}

	return w.prefetchNode(node)
}

func (w *Workspace) prefetchNode(node *Node) error {
	if err := node.ensureLoaded(); err != nil {
		return err
	}

	if node.IsDir() {
		for _, child := range node.Children() {
			if err := child.ensureLoaded(); err != nil {
				continue
			}
			if child.IsDir() {
				if err := w.prefetchNode(child); err != nil {
					continue
				}
			}
		}
	}

	return nil
}
