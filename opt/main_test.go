package opt

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"testing"
	"time"
)

const (
	TEST_LOOPS = 10000
	N          = float64(TEST_LOOPS)
)

var (
	corpus      = make([]fpelt, TEST_LOOPS)
	corpus2     = make([]fp2elt, TEST_LOOPS)
	corpus25519 = make([]fieldElement, TEST_LOOPS)
	corpusBig   = make([]*big.Int, TEST_LOOPS)
	pb          = fp2big(p)
)

func fp2big(x fpelt) *big.Int {
	x1b := big.NewInt(0).SetUint64(x[1])
	x1b = x1b.Lsh(x1b, 64)
	return x1b.Add(x1b, big.NewInt(0).SetUint64(x[0]))
}

func fpeq(x fpelt, y *big.Int) bool {
	xb := fp2big(x)
	return xb.Cmp(y) == 0
}

func randfp() (x fpelt) {
	x[0] = (uint64(rand.Uint32()) << 32) + uint64(rand.Uint32())
	x[1] = (uint64(rand.Uint32()) << 32) + uint64(rand.Uint32())
	return
}

func (x fpelt) String() string {
	return fmt.Sprintf("%016x%016x", x[1], x[0])
}

func testOp(f func(i int)) (min, max, mean, stdev float64) {
	m1 := float64(0)
	m2 := float64(0)
	min = 0
	max = 0
	for i := 0; i < TEST_LOOPS; i += 1 {
		tic := time.Now()
		f(i)
		toc := time.Now()
		net := float64(toc.Sub(tic))

		m1 += net
		m2 += net * net

		if min == 0 {
			min = net
		}
		min = math.Min(min, net)

		if max == 0 {
			max = net
		}
		max = math.Max(max, net)
	}

	mean = m1 / N
	variance := (m2 / N) - mean
	stdev = math.Sqrt(variance)
	return
}

func speedTest(label string, tests map[string]func(int)) {
	keys := []string{}
	for k, _ := range tests {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	template := "%5s - %10.2f   %10.2f   %10.2f   %10.2f\n"
	fmt.Println()
	fmt.Printf("===== Time for %d field operations (%s) =====\n", TEST_LOOPS, label)
	fmt.Printf("%5s - %10s   %10s   %10s   %10s\n", "", "min", "max", "mean", "stdev")
	for _, name := range keys {
		min, max, mean, stdev := testOp(tests[name])
		fmt.Printf(template, name, min, max, mean, stdev)
	}
	fmt.Println()
}

func TestMain(m *testing.M) {
	var buf [32]byte
	for i := range corpus25519 {
		rand.Read(buf[:])
		feFromBytes(&corpus25519[i], &buf)

		corpus[i] = randfp()
		fpreduce(&corpus[i])
		corpusBig[i] = fp2big(corpus[i])

		x := randfp()
		y := randfp()
		fpreduce(&x)
		fpreduce(&y)
		corpus2[i] = fp2elt{x, y}
	}

	m.Run()
}
