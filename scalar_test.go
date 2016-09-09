package curve4q

import (
	"testing"
)

func TestAbs(t *testing.T) {
	if abs(-15) != 15 {
		t.Fatalf("failed abs test (negative)")
	}

	if abs(15) != 15 {
		t.Fatalf("failed abs test (positive)")
	}
}

func TestSAddSub(t *testing.T) {
	x := scalar{0x8000000000000001, 0x8000000000000001, 0x8000000000000001, 0x0000000000000001}
	y := scalar{0x8000000000000001, 0x8000000000000001, 0x8000000000000001, 0x0000000000000001}
	z := scalar{0x0000000000000002, 0x0000000000000003, 0x0000000000000003, 0x0000000000000003}

	if z != sadd(x, y) {
		t.Fatalf("failed sadd test")
	}

	if x != ssub(z, y) {
		t.Fatalf("failed ssub test")
	}
}

func TestSSubI(t *testing.T) {
	x := scalar{0xffffffffffffffff, 0x0000000000000000, 0x0000000000000000, 0x0000000000000000}
	xp := scalar{0xfffffffffffffffe, 0x0000000000000000, 0x0000000000000000, 0x0000000000000000}
	xm := scalar{0x0000000000000000, 0x0000000000000001, 0x0000000000000000, 0x0000000000000000}

	if xp != ssubi(x, 1) {
		t.Fatalf("failed ssubi test (positive)")
	}

	if xm != ssubi(x, -1) {
		t.Fatalf("failed ssubi test (negative)")
	}
}

func TestSRsh4(t *testing.T) {
	x := scalar{0xffffffffffffffff, 0xeeeeeeeeeeeeeeee, 0xdddddddddddddddd, 0xcccccccccccccccc}
	y := scalar{0xefffffffffffffff, 0xdeeeeeeeeeeeeeee, 0xcddddddddddddddd, 0x0ccccccccccccccc}

	if y != srsh4(x) {
		t.Fatalf("failed srsh4 test")
	}
}

func TestSModN(t *testing.T) {
	z := scalar{0x0000000000000000, 0x0000000000000000, 0x0000000000000000, 0x0000000000000000}
	if z != smodN(N) {
		t.Fatalf("failed smodN test (N)")
	}

	// 16*N + 5
	x := scalar{0xfb2540ec7768ce75, 0xfbd004dfe0f79992, 0x05397829cbc14e5d, 0x029cbc14e5e0a72f}
	y := scalar{0x0000000000000005, 0x0000000000000000, 0x0000000000000000, 0x0000000000000000}
	if y != smodN(x) {
		t.Fatalf("failed smodN test ((N << 4) + 5)")
	}
}
