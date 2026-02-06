package utils

import (
	"hash/fnv"
	"sync"
)

// BloomFilter is a data structure used for efficient membership testing,
// providing high probability of false positives but low probability of false negatives.
type BloomFilter struct {
	bits      *customBitSet // The underlying bit set.
	size      uint          // The total number of bits in the filter.
	numHashes uint32        // The number of hash functions used.
	hashers   []hasherFunc  // A slice of hash functions to be used for hashing keys.
}

// hasherFunc is a function type that takes a byte slice and returns a 64-bit hash value.
type hasherFunc func([]byte) uint64

// newHasher creates a new hash function using a specific seed. The seed
// changes the output of the hashing process, allowing for different mappings
// of keys to bitset indices.
func newHasher(seed uint8) hasherFunc {
	return func(key []byte) uint64 {
		h := fnv.New64()      // provides a 64-bit hash value.
		h.Write([]byte{seed}) // Write the seed byte to initialize the hash function.
		h.Write(key)          // Write the key bytes to the hash function.
		return h.Sum64()      // Compute and return the resulting hash value as a 64-bit unsigned integer.
	}
}

// customBitSet is an internal structure that manages a set of bits efficiently using
// an array of uint64. This allows for fast insertion and testing operations.
type customBitSet struct {
	data []uint64 // // Array to store the bitset values, where each uint64 holds 64 bits
}

// Set marks the specified bit index as set in the bitset.
func (b *customBitSet) Set(i uint) {
	// We first calculate the byte index and bit offset within that byte. i >> 6 and i & 63
	byteIndex := i >> 6
	bitOffset := i & 63

	// Then we ark the specific bit at the calculated position by setting it to 1 using bitwise OR operation.
	b.data[byteIndex] |= (1 << bitOffset)
}

// Test checks if the specified bit index is set in the bitset.
func (b *customBitSet) Test(i uint) bool {
	// Calculate the byte index and bit offset within that byte.
	byteIndex := i >> 6
	bitOffset := i & 63

	// Check if the specific bit at the calculated position is set by performing a bitwise AND operation.
	return (b.data[byteIndex] & (1 << bitOffset)) != 0
}

// NewBloomFilter initializes a new Bloom Filter with the specified size and number of hash functions.
// The bitset is allocated to hold enough uint64s to cover the total number of bits, ensuring efficient storage.
func NewBloomFilter(size uint, numHashes int) *BloomFilter {
	// hasherFuncs based on the number of hashes requested.
	hashers := make([]hasherFunc, numHashes)
	// Initialize each hasher with a unique seed. The seed changes the output of the hash function.
	for i := range hashers {
		hashers[i] = newHasher(uint8(i))
	}
	// Return a new BloomFilter instance initialized with the calculated size, number of hashes,
	// and allocated bitset.
	return &BloomFilter{
		bits:      &customBitSet{data: make([]uint64, (size+63)/64)}, // We need enough uint64s to cover 'size' bits
		size:      size,
		numHashes: uint32(numHashes),
		hashers:   hashers,
	}
}

// Insert inserts a key into the Bloom Filter by hashing it multiple times with different hash functions
// and setting corresponding bits in the bitset.
func (bf *BloomFilter) Insert(key []byte) {
	// Iterate over each hasher and calculate the index for the key based on its hash value modulo the filter size.
	for _, hasher := range bf.hashers {
		index := uint(hasher(key)) % bf.size
		// Set the bit at the calculated index in the bitset.
		bf.bits.Set(index)
	}
}

// Contains checks if a key exists in the Bloom Filter by hashing it multiple times with different hash functions
// and verifying that all corresponding bits are set in the bitset.
func (bf *BloomFilter) Contains(key []byte) bool {
	// Iterate over each hasher and calculate the index for the key based on its hash value modulo the filter size.
	for _, hasher := range bf.hashers {
		index := uint(hasher(key)) % bf.size
		// Check if the bit at the calculated index is set in the bitset. If any bit is not set, return false.
		if !bf.bits.Test(index) {
			return false
		}
	}
	// If all bits are set, return true indicating that the key might be present in the filter.
	return true
}

// ConcurrentBloomFilter provides a thread-safe wrapper around the Bloom Filter for concurrent access.
type ConcurrentBloomFilter struct {
	mu    sync.RWMutex // Mutex to protect concurrent access to the Bloom Filter.
	bloom *BloomFilter // The underlying Bloom Filter instance being protected.
}

// NewConcurrentBloomFilter creates a new thread-safe Bloom Filter with the specified size and number of hash functions.
func NewConcurrentBloomFilter(size uint, numHashes int) *ConcurrentBloomFilter {
	// Create a new BloomFilter instance and wrap it in a ConcurrentBloomFilter.
	return &ConcurrentBloomFilter{
		bloom: NewBloomFilter(size, numHashes),
	}
}

// Insert inserts a key into the concurrent-safe Bloom Filter by locking the mutex for writing.
func (cbf *ConcurrentBloomFilter) Insert(key []byte) {
	cbf.mu.Lock()
	defer cbf.mu.Unlock()
	// Call the Insert method of the underlying Bloom Filter.
	cbf.bloom.Insert(key)
}

// Contains checks if a key exists in the concurrent-safe Bloom Filter by locking the mutex for reading.
func (cbf *ConcurrentBloomFilter) Contains(key []byte) bool {
	cbf.mu.RLock()
	defer cbf.mu.RUnlock()
	// Call the Contains method of the underlying Bloom Filter.
	return cbf.bloom.Contains(key)
}
