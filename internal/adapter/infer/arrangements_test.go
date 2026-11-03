package infer

import (
	"math/big"
	"slices"
	"testing"
)

// 🔴 The count behind every relation, put against the exact multinomial.
//
// The bar is all that stands between an agreement and a relation the manifest
// announces, and the count that meets it stops early. The exact answer is cheap
// with big integers, so every way of splitting up to twenty five records into
// groups, and every split of up to seventy records into two, is put to both, in
// the order the groups are counted and in the reverse. Below the bar the two
// have to be equal, and at the bar or above it the count has to say so.
func TestArrangementsMeetTheBarExactlyWhenTheMultinomialDoes(t *testing.T) {
	bar := big.NewInt(int64(Coincidences))
	check := func(sizes []int) {
		for _, order := range [][]int{sizes, reversed(sizes)} {
			exact := multinomial(order)
			got := arrangements(order)
			switch {
			case exact.Cmp(bar) >= 0 && got != Coincidences:
				t.Errorf("groups of %v can be dealt out %s ways, at the bar or past it, and the "+
					"count said %.0f", order, exact, got)
			case exact.Cmp(bar) < 0 && got != float64(exact.Int64()):
				t.Errorf("groups of %v can be dealt out %s ways, under the bar, and the count "+
					"said %.0f", order, exact, got)
			}
		}
	}
	for total := 1; total <= 25; total++ {
		splits(total, total, nil, check)
	}
	for total := 26; total <= 70; total++ {
		check([]int{total})
		for fewer := 1; fewer <= total/2; fewer++ {
			check([]int{fewer, total - fewer})
		}
	}
}

// splits hands every way of writing total as a sum of parts no larger than
// largest to visit, the parts from the smallest up as sizesOf orders them.
func splits(total, largest int, parts []int, visit func([]int)) {
	if total == 0 {
		visit(reversed(parts))
		return
	}
	for part := min(total, largest); part >= 1; part-- {
		splits(total-part, part, append(parts, part), visit)
	}
}

func reversed(sizes []int) []int {
	turned := slices.Clone(sizes)
	slices.Reverse(turned)
	return turned
}

// multinomial is the number of ways of dealing out groups of these sizes,
// worked out in full: the factorial of the total over the factorial of each.
func multinomial(sizes []int) *big.Int {
	total := 0
	for _, size := range sizes {
		total += size
	}
	ways := new(big.Int).MulRange(1, int64(total))
	for _, size := range sizes {
		ways.Quo(ways, new(big.Int).MulRange(1, int64(size)))
	}
	return ways
}
