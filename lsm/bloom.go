package lsm

import (
	"encoding/json"
	"hash/fnv"
	"math"
	"os"
)

type BloomFilter struct {
	Bitset []bool
	K      int
	M      int
	Size   int
}

func NewBloomFilter(n int, falsePositiveRate float64) *BloomFilter {
	m := int(-float64(n) * math.Log(falsePositiveRate) / math.Pow(math.Log(2), 2)) // m = -n * ln(p) / (ln(2)^2)
	k := int(float64(m) / float64(n) * math.Log(2))                                // k = m/n * ln(2)

	if m < 1 {
		m = 1
	}
	if k < 1 {
		k = 1
	}

	return &BloomFilter{
		Bitset: make([]bool, m),
		K:      k,
		M:      m,
	}
}

func (bf *BloomFilter) Add(key string) {
	for i := 0; i < bf.K; i++ {
		position := bf.hash(key, i) % bf.M
		if position < 0 {
			position = -position
		}
		bf.Bitset[position] = true
	}
	bf.Size++
}

func (bf *BloomFilter) Check(key string) bool {
	for i := 0; i < bf.K; i++ {
		position := bf.hash(key, i) % bf.M
		if position < 0 {
			position = -position
		}
		if !bf.Bitset[position] {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) hash(key string, i int) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	h.Write([]byte{byte(i)})
	return int(h.Sum32())
}

func SaveBloomFilter(filename string, bf *BloomFilter) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(bf)
}

func LoadBloomFilter(filename string) (*BloomFilter, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var bf BloomFilter
	if err := json.NewDecoder(file).Decode(&bf); err != nil {
		return nil, err
	}
	return &bf, nil
}
