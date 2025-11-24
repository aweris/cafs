package cafs

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

// Snapshot represents an immutable point-in-time view of a filesystem.
// All content is content-addressed and cannot be modified.
type Snapshot struct {
	rootHash string
	store    Store

	cache map[string]*Node
	mu    sync.RWMutex
}

// newSnapshot creates a new snapshot from a root hash.
func newSnapshot(rootHash string, store Store) *Snapshot {
	return &Snapshot{
		rootHash: rootHash,
		store:    store,
		cache:    make(map[string]*Node),
	}
}

// RootHash returns the content hash of the root tree.
func (s *Snapshot) RootHash() string {
	return s.rootHash
}

func (s *Snapshot) Open(name string) (fs.File, error) {
	// TODO: Implement
	return nil, fs.ErrNotExist
}

func (s *Snapshot) ReadFile(path string) ([]byte, error) {
	// Don't hold lock during navigation (loadNode needs to acquire lock)
	node, err := s.navigateFromHash(s.rootHash, path)
	if err != nil {
		return nil, err
	}

	if node.IsDir() {
		return nil, fmt.Errorf("%s: is a directory", path)
	}

	return node.Content(), nil
}

func (s *Snapshot) ReadDir(name string) ([]fs.DirEntry, error) {
	// TODO: Implement
	return nil, fs.ErrNotExist
}

func (s *Snapshot) Stat(name string) (fs.FileInfo, error) {
	// TODO: Implement
	return nil, fs.ErrNotExist
}

// loadNode loads a node from store by hash (with caching).
func (s *Snapshot) loadNode(hash string) (*Node, error) {
	s.mu.RLock()
	if node, ok := s.cache[hash]; ok {
		s.mu.RUnlock()
		return node, nil
	}
	s.mu.RUnlock()

	data, err := s.store.Get(context.TODO(), hash)
	if err != nil {
		return nil, fmt.Errorf("failed to load object %s: %w", hash, err)
	}

	node, err := s.decodeObject(data, hash)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[hash] = node
	s.mu.Unlock()

	return node, nil
}

func (s *Snapshot) decodeObject(data []byte, hash string) (*Node, error) {
	idx := bytes.IndexByte(data, 0)
	if idx == -1 {
		return nil, fmt.Errorf("invalid object: missing null terminator")
	}

	header := string(data[:idx])
	content := data[idx+1:]

	if strings.HasPrefix(header, "blob ") {
		node := &Node{
			content: content,
			mode:    0644,
			size:    int64(len(content)),
			hash:    hash,
			dirty:   false,
			store:   s.store,
		}
		return node, nil
	}

	if strings.HasPrefix(header, "tree ") {
		entries, err := s.decodeTreeEntries(content)
		if err != nil {
			return nil, err
		}

		node := &Node{
			mode:     fs.ModeDir | 0755,
			children: make(map[string]*Node),
			hash:     hash,
			dirty:    false,
			store:    s.store,
			loaded:   true,
		}

		for _, entry := range entries {
			childHash := hex.EncodeToString(entry.Hash[:])
			child := &Node{
				name:     entry.Name,
				mode:     entry.Mode,
				hash:     childHash,
				loaded:   false,
				store:    s.store,
				parent:   node, // Set parent pointer
				children: make(map[string]*Node), // Initialize for directories
			}
			node.children[entry.Name] = child
		}

		return node, nil
	}

	return nil, fmt.Errorf("unknown object type: %s", header)
}

func (s *Snapshot) decodeTreeEntries(data []byte) ([]treeEntry, error) {
	var entries []treeEntry
	reader := bytes.NewReader(data)

	for reader.Len() > 0 {
		var entry treeEntry

		var mode uint32
		if err := binary.Read(reader, binary.BigEndian, &mode); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		entry.Mode = fs.FileMode(mode)

		if _, err := io.ReadFull(reader, entry.Hash[:]); err != nil {
			return nil, err
		}

		var nameLen uint16
		if err := binary.Read(reader, binary.BigEndian, &nameLen); err != nil {
			return nil, err
		}

		nameBuf := make([]byte, nameLen)
		if _, err := io.ReadFull(reader, nameBuf); err != nil {
			return nil, err
		}
		entry.Name = string(nameBuf)

		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *Snapshot) navigateFromHash(rootHash, path string) (*Node, error) {
	root, err := s.loadNode(rootHash)
	if err != nil {
		return nil, err
	}

	path = filepath.Clean("/" + path)
	if path == "/" {
		return root, nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := root

	for _, part := range parts {
		if part == "" {
			continue
		}

		if !current.IsDir() {
			return nil, fs.ErrNotExist
		}

		if err := current.ensureLoaded(); err != nil {
			return nil, err
		}

		child, ok := current.children[part]
		if !ok {
			return nil, fs.ErrNotExist
		}

		if !child.loaded && child.hash != "" {
			child, err = s.loadNode(child.hash)
			if err != nil {
				return nil, err
			}
		}

		current = child
	}

	return current, nil
}
