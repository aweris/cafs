package remote

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

const (
	LayerTargetSize = 5 * 1024 * 1024  // 5MB target
	LayerMinSize    = 2 * 1024 * 1024  // 2MB minimum before combining
	LayerSoftMax    = 10 * 1024 * 1024 // 10MB soft maximum
	digestLen       = 71               // "sha256:" (7) + hex (64)
)

type PrefixInfo struct {
	Hash  string `json:"hash"`
	Layer string `json:"layer"`
}

func GroupByPrefix(objects map[string][]byte) map[string]map[string][]byte {
	result := make(map[string]map[string][]byte)
	for digest, data := range objects {
		prefix := extractPrefix(digest)
		if result[prefix] == nil {
			result[prefix] = make(map[string][]byte)
		}
		result[prefix][digest] = data
	}
	return result
}

func extractPrefix(digest string) string {
	if rest, ok := strings.CutPrefix(digest, "sha256:"); ok && len(rest) >= 2 {
		return rest[:2]
	}
	if len(digest) >= 2 {
		return digest[:2]
	}
	return "00"
}

func PrefixHash(blobs map[string][]byte) string {
	if len(blobs) == 0 {
		return ""
	}

	digests := make([]string, 0, len(blobs))
	for d := range blobs {
		digests = append(digests, d)
	}
	sort.Strings(digests)

	h := sha256.New()
	for _, d := range digests {
		h.Write([]byte(d))
		binary.Write(h, binary.BigEndian, int64(len(blobs[d])))
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func PrefixSize(blobs map[string][]byte) int64 {
	var total int64
	for _, data := range blobs {
		total += int64(len(data))
	}
	return total
}

// PackLayer packs blobs into binary format: [digest 71B][length 8B][data]...
func PackLayer(blobs map[string][]byte) []byte {
	digests := make([]string, 0, len(blobs))
	for d := range blobs {
		digests = append(digests, d)
	}
	sort.Strings(digests)

	var buf bytes.Buffer
	digestBuf := make([]byte, digestLen)
	lenBuf := make([]byte, 8)

	for _, digest := range digests {
		data := blobs[digest]

		// Write fixed-size digest (padded with zeros)
		copy(digestBuf, digest)
		for i := len(digest); i < digestLen; i++ {
			digestBuf[i] = 0
		}
		buf.Write(digestBuf)

		// Write length
		binary.BigEndian.PutUint64(lenBuf, uint64(len(data)))
		buf.Write(lenBuf)

		// Write data
		buf.Write(data)
	}
	return buf.Bytes()
}

func UnpackLayer(data []byte) (map[string][]byte, error) {
	result := make(map[string][]byte)
	buf := bytes.NewReader(data)
	digestBuf := make([]byte, digestLen)

	for buf.Len() > 0 {
		if _, err := buf.Read(digestBuf); err != nil {
			return nil, fmt.Errorf("read digest: %w", err)
		}
		digest := strings.TrimRight(string(digestBuf), "\x00")

		var length uint64
		if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
			return nil, fmt.Errorf("read length: %w", err)
		}

		blobData := make([]byte, length)
		if _, err := buf.Read(blobData); err != nil {
			return nil, fmt.Errorf("read data: %w", err)
		}

		result[digest] = blobData
	}

	return result, nil
}

func BuildLayerPlan(prefixSizes map[string]int64) [][]string {
	prefixes := make([]string, 0, len(prefixSizes))
	for p := range prefixSizes {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	var layers [][]string
	var current []string
	var size int64

	for _, prefix := range prefixes {
		prefixSize := prefixSizes[prefix]

		if len(current) == 0 {
			current = append(current, prefix)
			size = prefixSize
			continue
		}

		newSize := size + prefixSize
		if newSize <= LayerSoftMax {
			current = append(current, prefix)
			size = newSize
		} else if size < LayerMinSize && newSize <= 2*LayerSoftMax {
			current = append(current, prefix)
			size = newSize
		} else {
			layers = append(layers, current)
			current = []string{prefix}
			size = prefixSize
		}
	}

	if len(current) > 0 {
		layers = append(layers, current)
	}

	return layers
}

func CollectPrefixBlobs(prefixes []string, byPrefix map[string]map[string][]byte) map[string][]byte {
	result := make(map[string][]byte)
	for _, prefix := range prefixes {
		for digest, data := range byPrefix[prefix] {
			result[digest] = data
		}
	}
	return result
}

func CalculatePrefixSizes(byPrefix map[string]map[string][]byte) map[string]int64 {
	result := make(map[string]int64)
	for prefix, blobs := range byPrefix {
		result[prefix] = PrefixSize(blobs)
	}
	return result
}
