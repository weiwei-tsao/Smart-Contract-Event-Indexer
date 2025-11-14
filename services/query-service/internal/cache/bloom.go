package cache

import (
	"encoding/binary"
	"hash/fnv"
	"sync"
)

// BloomFilter provides a simple probabilistic membership filter to guard cache misses.
type BloomFilter struct {
	size   uint64
	hashes uint64
	bits   []uint64
	mu     sync.RWMutex
}

// NewBloomFilter initializes a Bloom filter with the provided bit size and hash count.
func NewBloomFilter(size uint64, hashes uint64) *BloomFilter {
	if size == 0 {
		size = 1 << 20 // fall back to 1M bits
	}
	if hashes == 0 {
		hashes = 3
	}

	words := (size + 63) / 64
	return &BloomFilter{
		size:   size,
		hashes: hashes,
		bits:   make([]uint64, words),
	}
}

// Add inserts the provided data into the filter.
func (bf *BloomFilter) Add(data []byte) {
	if bf == nil || bf.size == 0 {
		return
	}

	positions := bf.hashPositions(data)

	bf.mu.Lock()
	defer bf.mu.Unlock()

	for _, pos := range positions {
		word := pos / 64
		bit := pos % 64
		bf.bits[word] |= 1 << bit
	}
}

// ProbablyContains indicates whether the provided data might be present.
func (bf *BloomFilter) ProbablyContains(data []byte) bool {
	if bf == nil || bf.size == 0 {
		return false
	}

	positions := bf.hashPositions(data)

	bf.mu.RLock()
	defer bf.mu.RUnlock()

	for _, pos := range positions {
		word := pos / 64
		bit := pos % 64
		if bf.bits[word]&(1<<bit) == 0 {
			return false
		}
	}

	return true
}

func (bf *BloomFilter) hashPositions(data []byte) []uint64 {
	positions := make([]uint64, bf.hashes)
	seed := bf.baseHash(data)
	salt := bf.secondaryHash(data)

	for i := uint64(0); i < bf.hashes; i++ {
		hash := seed + i*salt + uint64(i*i)
		positions[i] = hash % bf.size
	}

	return positions
}

func (bf *BloomFilter) baseHash(data []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(data)
	return h.Sum64()
}

func (bf *BloomFilter) secondaryHash(data []byte) uint64 {
	h := fnv.New64()
	_, _ = h.Write(data)

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, h.Sum64())
	return h.Sum64() | 1 // ensure odd to avoid duplicates
}
