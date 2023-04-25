package controllers

import "math/big"

// ZX81 standard
// More: https://en.wikipedia.org/wiki/Linear_congruential_generator
const (
	a = 75
	c = 74
	m = 65537 // 2^16 + 1
)

type rnd struct {
	value *big.Int
}

func newRnd(val *big.Int) *rnd {
	return &rnd{value: val}
}

// getNext returns the next random value
func (r *rnd) next() *big.Int {
	val := new(big.Int).Add(new(big.Int).Mul(r.value, big.NewInt(a)), big.NewInt(c))
	res := new(big.Int).Mod(val, big.NewInt(m))
	r.value = res
	return r.value
}

// getIndex returns the next list index based on current list length
func getIndex(val *big.Int, ln int) int {
	return int(new(big.Int).Mod(val, big.NewInt(int64(ln))).Int64())
}
