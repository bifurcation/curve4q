package curve4q

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func randfp() (x fpelt) {
	x[0] = (uint64(rand.Uint32()) << 32) + uint64(rand.Uint32())
	x[1] = (uint64(rand.Uint32()) << 32) + uint64(rand.Uint32())
	return
}

func fp2big(x fpelt) *big.Int {
	x1b := big.NewInt(0).SetUint64(x[1])
	x1b = x1b.Lsh(x1b, 64)
	return x1b.Add(x1b, big.NewInt(0).SetUint64(x[0]))
}

func fpeq(x fpelt, y *big.Int) bool {
	xb := fp2big(x)
	return xb.Cmp(y) == 0
}

func (x fpelt) String() string {
	return fmt.Sprintf("%016x%016x", x[1], x[0])
}

const (
	TEST_LOOPS = 1000
)

var (
	pb        = fp2big(p)
	corpus    = make([]fpelt, TEST_LOOPS)
	bigCorpus = make([]*big.Int, TEST_LOOPS)
	corpus2   = make([]fp2elt, TEST_LOOPS)
)

func TestMain(m *testing.M) {
	for i := range corpus {
		corpus[i] = randfp()
		fpreduce(&corpus[i])
		bigCorpus[i] = fp2big(corpus[i])

		x := randfp()
		y := randfp()
		fpreduce(&x)
		fpreduce(&y)
		corpus2[i] = fp2elt{x, y}
	}

	m.Run()
}

func TestPerf(t *testing.T) {
	speedtest := func(f func(i int)) time.Duration {
		tic := time.Now()
		for i := 0; i < TEST_LOOPS; i += 1 {
			f(i)
		}
		toc := time.Now()
		return toc.Sub(tic)
	}

	tests := map[string]func(int){
		"add": func(i int) { fp2add(corpus2[i], corpus2[(i+1)%TEST_LOOPS]) },
		"mul": func(i int) { fp2mul(corpus2[i], corpus2[(i+1)%TEST_LOOPS]) },
		"sqr": func(i int) { fp2sqr(corpus2[i]) },
		"inv": func(i int) { fp2inv(corpus2[i]) },
	}

	fmt.Println()
	fmt.Printf("===== Time for %d field operations =====\n", TEST_LOOPS)
	for name := range tests {
		t1271 := speedtest(tests[name])
		fmt.Printf("%5s %v\n", name, t1271)
	}
	fmt.Println()

	P1 := _AffineToR1(affine{Gx, Gy})
	clearCounters()
	tic := time.Now()
	dbl(P1)
	toc := time.Now()
	fmt.Printf("dbl: M=%d S=%d A=%d I=%d t=%v\n", fp2M, fp2S, fp2A, fp2I, toc.Sub(tic))

	m := randScalar()
	P2 := affine{Gx, Gy}
	clearCounters()
	tic = time.Now()
	dhWindowed(m, P2, nil)
	toc = time.Now()
	fmt.Printf("dh-win: M=%d S=%d A=%d I=%d t=%v\n", fp2M, fp2S, fp2A, fp2I, toc.Sub(tic))

	m = randScalar()
	P2 = affine{Gx, Gy}
	clearCounters()
	tic = time.Now()
	dhEndo(m, P2, nil)
	toc = time.Now()
	fmt.Printf("dh-endo: M=%d S=%d A=%d I=%d t=%v\n", fp2M, fp2S, fp2A, fp2I, toc.Sub(tic))
}

func TestFPSelect(t *testing.T) {
	x := fpelt{1, 2}
	y := fpelt{3, 4}

	z := fpselect(1, x, y)
	if z != x {
		t.Fatalf("fpselect with c=1 failed %v", z)
	}

	z = fpselect(0, x, y)
	if z != y {
		t.Fatalf("fpselect with c=0 failed %v", z)
	}
}

func TestFPReduce(t *testing.T) {
	for _ = range corpus {
		x := randfp()
		before := fp2big(x)

		fpreduce(&x)
		after := before.Mod(before, pb)

		if !fpeq(x, after) {
			t.Fatalf("fpreduce failed %s != %s", after.Text(16), x)
		}
	}
}

func TestFPAdd(t *testing.T) {
	for i := range corpus {
		x := corpus[i]
		y := corpus[(i+1)%TEST_LOOPS]
		z := fpadd(x, y)

		xb := bigCorpus[i]
		yb := bigCorpus[(i+1)%TEST_LOOPS]
		zb := big.NewInt(0).Add(xb, yb)
		zb.Mod(zb, pb)

		if !fpeq(z, zb) {
			t.Fatalf("fpadd failed [%d] %s != %s", i, zb.Text(16), z)
		}
	}
}

func TestFPNeg(t *testing.T) {
	zero := fpelt{0, 0}
	if fpneg(zero) != zero {
		t.Fatalf("fpneg failed to handle zero %v", fpneg(zero))
	}
	return

	for i := range corpus {
		x := corpus[i]
		z := fpneg(x)

		xb := bigCorpus[i]
		zb := big.NewInt(0).Sub(pb, xb)
		zb.Mod(zb, pb)

		if !fpeq(z, zb) {
			t.Fatalf("fpneg failed [%d] %s != %s", i, zb.Text(16), z)
		}
	}
}

func TestFPSub(t *testing.T) {
	for i := range corpus {
		x := corpus[i]
		y := corpus[(i+1)%TEST_LOOPS]
		z := fpsub(x, y)

		xb := bigCorpus[i]
		yb := bigCorpus[(i+1)%TEST_LOOPS]
		zb := big.NewInt(0).Sub(xb, yb)
		zb.Mod(zb, pb)

		if !fpeq(z, zb) {
			t.Fatalf("fpadd failed [%d] %s != %s", i, zb.Text(16), z)
		}
	}
}

func TestFPMul(t *testing.T) {
	for i := range corpus {
		x := corpus[i]
		y := corpus[(i+1)%TEST_LOOPS]
		z := fpmul(x, y)

		xb := bigCorpus[i]
		yb := bigCorpus[(i+1)%TEST_LOOPS]
		zb := big.NewInt(0).Mul(xb, yb)
		zb.Mod(zb, pb)

		if !fpeq(z, zb) {
			t.Fatalf("fpmul failed %s != %s", zb.Text(16), z)
		}
	}
}

func TestFPSqr(t *testing.T) {
	for i := range corpus {
		x := corpus[i]
		z := fpsqr(x)

		xb := bigCorpus[i]
		zb := big.NewInt(0).Mul(xb, xb)
		zb.Mod(zb, pb)

		if !fpeq(z, zb) {
			t.Fatalf("fpneg failed [%d] %s != %s", i, zb.Text(16), z)
		}
	}
}

func TestFPInv(t *testing.T) {
	for i := range corpus {
		x := corpus[i]
		inv := fpinv(x)
		z := fpmul(x, inv)
		if z[0] != 1 || z[1] != 0 {
			t.Fatalf("fpinv did not produce inverse [%d] %s != %s", i, inv, z)
		}

		xb := bigCorpus[i]
		invb := fp2big(inv)
		zb := big.NewInt(0).Mul(xb, invb)
		zb.Mod(zb, pb)

		if zb.Uint64() != uint64(1) {
			t.Fatalf("fpneg failed [%d] %s != %s", i, zb.Text(16), z)
		}
	}
}

func TestFPInvSqrt(t *testing.T) {
	for i := range corpus {
		x := corpus[i]
		inv := fpinvsqrt(x)
		z := fpmul(fpmul(x, inv), inv)
		zIsOne := z[0] != 1 || z[1] != 0
		zIsMinusOne := z[0] != p0-1 || z[1] != p1
		if !zIsOne && !zIsMinusOne {
			t.Fatalf("fpinvsqrt did not produce inverse [%d] %s != %s", i, inv, z)
		}

		xb := bigCorpus[i]
		invb := fp2big(inv)
		zb := big.NewInt(0).Mul(xb, invb)
		zb.Mul(zb, invb)
		zb.Mod(zb, pb)

		one := big.NewInt(1)
		pm1 := big.NewInt(0).Sub(pb, one)

		if zb.Cmp(one) != 0 && zb.Cmp(pm1) != 0 {
			t.Fatalf("fpinvsqrt failed [%d] %x", i, zb.Abs(zb).Uint64())
		}
	}
}
