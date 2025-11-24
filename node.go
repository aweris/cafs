package cafs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
	"time"
)

// Node represents a single filesystem entry (file or directory) in memory.
// Nodes form a tree structure where directories contain child nodes.
type Node struct {
	name    string
	mode    fs.FileMode
	modTime time.Time
	size    int64

	content  []byte
	children map[string]*Node

	hash  string // Computed lazily on Push
	dirty bool

	parent *Node

	loaded bool // For lazy-loaded snapshot nodes
	store  Store
}

// newFileNode creates a new file node.
func newFileNode(name string, content []byte, mode fs.FileMode) *Node {
	return &Node{
		name:    name,
		mode:    mode,
		modTime: time.Now(),
		size:    int64(len(content)),
		content: content,
		dirty:   true,
	}
}

// newDirNode creates a new directory node.
func newDirNode(name string, mode fs.FileMode) *Node {
	return &Node{
		name:     name,
		mode:     mode | fs.ModeDir,
		modTime:  time.Now(),
		children: make(map[string]*Node),
		dirty:    true,
	}
}

// IsDir returns true if this is a directory.
func (n *Node) IsDir() bool {
	return n.mode.IsDir()
}

// Name returns the node name.
func (n *Node) Name() string {
	return n.name
}

// Mode returns the file mode.
func (n *Node) Mode() fs.FileMode {
	return n.mode
}

// Size returns the file size (0 for directories).
func (n *Node) Size() int64 {
	if n.IsDir() {
		return 0
	}
	if n.size == 0 && !n.loaded && n.hash != "" && n.store != nil {
		if err := n.ensureLoaded(); err != nil {
			return 0
		}
	}
	return n.size
}

// ModTime returns the modification time.
func (n *Node) ModTime() time.Time {
	return n.modTime
}

// Content returns the file content (nil for directories).
func (n *Node) Content() []byte {
	return n.content
}

// Children returns the directory children (nil for files).
func (n *Node) Children() map[string]*Node {
	return n.children
}

// Hash returns the content hash (computed lazily).
func (n *Node) Hash() string {
	return n.hash
}

// IsDirty returns true if the node needs hash recomputation.
func (n *Node) IsDirty() bool {
	return n.dirty
}

// loadBlobSize fetches blob to determine size and caches it for later reads.
func (n *Node) loadBlobSize(hash string) (int64, error) {
	data, err := n.store.Get(context.TODO(), hash)
	if err != nil {
		return 0, err
	}

	idx := bytes.IndexByte(data, 0)
	if idx == -1 {
		return 0, fmt.Errorf("invalid blob format")
	}

	header := string(data[:idx])
	if !strings.HasPrefix(header, "blob ") {
		return 0, fmt.Errorf("not a blob")
	}

	content := data[idx+1:]
	return int64(len(content)), nil
}

// ensureLoaded loads node content from store if not already loaded.
// This implements lazy loading for snapshot nodes.
func (n *Node) ensureLoaded() error {
	if n.loaded {
		return nil
	}

	if n.hash == "" || n.store == nil {
		// Nothing to load
		n.loaded = true
		return nil
	}

	data, err := n.store.Get(context.TODO(), n.hash)
	if err != nil {
		return fmt.Errorf("failed to load node: %w", err)
	}

	idx := bytes.IndexByte(data, 0)
	if idx == -1 {
		return fmt.Errorf("invalid object format")
	}

	header := string(data[:idx])
	content := data[idx+1:]

	if strings.HasPrefix(header, "blob ") {
		n.content = content
		n.size = int64(len(content))
	} else if strings.HasPrefix(header, "tree ") {
		entries, err := decodeTreeData(content)
		if err != nil {
			return err
		}

		n.children = make(map[string]*Node)
		for _, entry := range entries {
			childHash := hex.EncodeToString(entry.Hash[:])
			child := &Node{
				name:   entry.Name,
				mode:   entry.Mode,
				hash:   childHash,
				loaded: false,
				store:  n.store,
				parent: n,
			}

			if !entry.IsDir && child.hash != "" && n.store != nil {
				size, err := n.loadBlobSize(child.hash)
				if err == nil {
					child.size = size
				}
			}

			n.children[entry.Name] = child
		}
	}

	n.loaded = true
	return nil
}

// decodeTreeData decodes tree entries from binary data.
func decodeTreeData(data []byte) ([]treeEntry, error) {
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

// computeHash computes the content hash for this node.
// For directories, this recursively computes child hashes first.
func (n *Node) computeHash(store Store) (string, error) {
	if !n.dirty && n.hash != "" {
		return n.hash, nil
	}

	if !n.IsDir() {
		hash, encoded, err := encodeBlob(n.content)
		if err != nil {
			return "", err
		}

		if store != nil {
			if _, err := store.Put(context.TODO(), encoded); err != nil {
				return "", err
			}
		}

		n.hash = hash
		n.dirty = false
		return hash, nil
	}

	var entries []treeEntry
	for name, child := range n.children {
		childHash, err := child.computeHash(store)
		if err != nil {
			return "", err
		}

		var h [32]byte
		decoded, err := decodeHash(childHash)
		if err != nil {
			return "", err
		}
		copy(h[:], decoded)

		entries = append(entries, treeEntry{
			Name:  name,
			Mode:  child.mode,
			Hash:  h,
			IsDir: child.IsDir(),
		})
	}

	hash, encoded, err := encodeTree(entries)
	if err != nil {
		return "", err
	}

	if store != nil {
		if _, err := store.Put(context.TODO(), encoded); err != nil {
			return "", err
		}
	}

	n.hash = hash
	n.dirty = false
	return hash, nil
}

// getChild returns a child node by name.
func (n *Node) getChild(name string) (*Node, bool) {
	if !n.IsDir() {
		return nil, false
	}
	child, ok := n.children[name]
	return child, ok
}

// addChild adds a child node.
func (n *Node) addChild(name string, child *Node) {
	if !n.IsDir() {
		return
	}
	// Ensure children map is initialized (for lazy-loaded nodes)
	if n.children == nil {
		n.children = make(map[string]*Node)
	}
	child.parent = n
	n.children[name] = child
	n.markDirty()
}

// markDirty marks this node and all parents as dirty.
func (n *Node) markDirty() {
	n.dirty = true
	n.modTime = time.Now()
	if n.parent != nil {
		n.parent.markDirty()
	}
}

// setContent updates file content.
// Only marks as dirty if content actually changed.
func (n *Node) setContent(content []byte) {
	if n.IsDir() {
		return
	}

	if !bytes.Equal(n.content, content) {
		n.content = content
		n.size = int64(len(content))
		n.markDirty()
	}
}

// treeEntry represents an entry in a tree (internal representation).
type treeEntry struct {
	Name  string
	Mode  fs.FileMode
	Hash  [32]byte
	IsDir bool
}

// encodeBlob encodes file content as a blob object.
// Format: "blob {size}\0{content}" → SHA256
func encodeBlob(content []byte) (hash string, encoded []byte, err error) {
	header := fmt.Sprintf("blob %d\x00", len(content))
	buf := make([]byte, len(header)+len(content))
	copy(buf, header)
	copy(buf[len(header):], content)

	h := sha256.Sum256(buf)
	hash = hex.EncodeToString(h[:])

	return hash, buf, nil
}

// encodeTree encodes directory entries as a tree object.
// Format: "tree {size}\0{entries}"
// Entry format: {mode:4bytes}{hash:32bytes}{nameLen:2bytes}{name}
func encodeTree(entries []treeEntry) (hash string, encoded []byte, err error) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	var entriesBuf bytes.Buffer
	for _, entry := range entries {
		binary.Write(&entriesBuf, binary.BigEndian, uint32(entry.Mode))
		entriesBuf.Write(entry.Hash[:])
		binary.Write(&entriesBuf, binary.BigEndian, uint16(len(entry.Name)))
		entriesBuf.WriteString(entry.Name)
	}

	entriesData := entriesBuf.Bytes()
	header := fmt.Sprintf("tree %d\x00", len(entriesData))
	buf := make([]byte, len(header)+len(entriesData))
	copy(buf, header)
	copy(buf[len(header):], entriesData)

	h := sha256.Sum256(buf)
	hash = hex.EncodeToString(h[:])

	return hash, buf, nil
}

// decodeHash decodes a hex-encoded hash.
func decodeHash(hexHash string) ([]byte, error) {
	return hex.DecodeString(hexHash)
}
