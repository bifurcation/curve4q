package curve4q

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"testing"
)

/********** Utilities **********/

var (
	G1 = r1{Gx, Gy, fp2One, Gx, Gy}
	O1 = r1{Ox, Oy, fp2One, Ox, Oy}
)

func toScalar(x uint64) scalar {
	return scalar{x, 0, 0, 0}
}

func randScalar() scalar {
	return scalar{
		uint64(rand.Int63()),
		uint64(rand.Int63()),
		uint64(rand.Int63()),
		uint64(rand.Int63()),
	}
}

/**********/

func TestPointOnCurve(t *testing.T) {
	if !pointOnCurve(Ox, Oy) {
		t.Fatalf("Neutral point not on the curve")
	}

	if !pointOnCurve(Gx, Gy) {
		t.Fatalf("Base point not on the curve")
	}
}

func TestEncodeDecode(t *testing.T) {
	GEncHex := "87b2cb2b46a224b95a7820a19bee3f0e5c8b4c8444c3a74942020e63f84a1c6e"
	GEnc, _ := hex.DecodeString(GEncHex)

	enc := encode(affine{Gx, Gy})
	if !bytes.Equal(enc, GEnc) {
		t.Fatalf("Encode test failed")
	}

	dec := decode(GEnc)
	if dec.X != Gx || dec.Y != Gy {
		t.Fatalf("Decode test failed")
	}
}

func TestReps(t *testing.T) {
	x := fp2elt{fpint(0), fpint(1)}
	x2 := fp2elt{fpint(0), fpint(2)}
	y := fp2elt{fpint(2), fpint(0)}
	y2 := fp2elt{fpint(4), fpint(0)}
	xy := fp2elt{fpint(2), fpint(1)}
	yx := fp2elt{fpint(2), fpelt{p0 - 1, p1}}
	z := fp2elt{fpint(3), fpint(4)}
	z2 := fp2elt{fpint(6), fpint(8)}
	ta := fp2elt{fpint(5), fpint(0)}
	tb := fp2elt{fpint(1), fpint(6)}
	tt := fp2elt{fpint(5), fpint(30)}
	t2 := fp2add(tt, tt)
	td2 := fp2mul(d, t2)

	r1pt := r1{x, y, z, ta, tb}
	r2pt := r2{xy, yx, z2, td2}
	r3pt := r3{xy, yx, z, tt}
	r1pt2 := r1{x2, y2, z2, t2, fp2One}

	if _R1toR2(r1pt) != r2pt {
		t.Fatalf("failed R1toR2")
	}

	if _R1toR3(r1pt) != r3pt {
		t.Fatalf("failed R1toR3")
	}

	if _R2toR1(r2pt) != r1pt2 {
		t.Fatalf("failed R2toR4")
	}
}

func TestCore(t *testing.T) {
	// Test doubling
	A := _AffineToR1(affine{Gx, Gy})
	for i := 0; i < 1000; i += 1 {
		A = dbl(A)
	}
	doubleP := affine{
		fp2elt{
			fpelt{0xC9099C54855859D6, 0x2C3FD8822C82270F},
			fpelt{0xA7B3F6E2043E8E68, 0x4DA5B9E83AA7A1B2},
		},
		fp2elt{
			fpelt{0x3EE089F0EB49AA14, 0x2001EB3A57688396},
			fpelt{0x1FEE5617A7E954CD, 0x0FFDB0D761421F50},
		},
	}
	if _R1toAffine(A) != doubleP {
		t.Fatalf("failed DBL test")
	}

	// Test that the neutral element is neutral
	G := _AffineToR1(affine{Gx, Gy})
	O := _AffineToR1(affine{Ox, Oy})
	GO := add(G, _R1toR2(O))
	OG := add(O, _R1toR2(G))
	if _R1toAffine(GO) != _R1toAffine(G) {
		t.Fatalf("failed right-neutral test")
	}
	if _R1toAffine(OG) != _R1toAffine(G) {
		t.Fatalf("failed left-neutral test")
	}

	// Test point doubling by addition
	A = G
	for i := 0; i < 1000; i += 1 {
		A = add(A, _R1toR2(A))
	}
	if _R1toAffine(A) != doubleP {
		t.Fatalf("failed double-by-add test")
	}

	// Test repeated addition of the same point
	A = G
	B := _R1toR2(A)
	A = dbl(A)
	for i := 0; i < 1000; i += 1 {
		A = add(A, B)
	}
	A1000 := affine{
		fp2elt{
			fpelt{0x6480B1EF0A151DB0, 0x3E243958590C4D90},
			fpelt{0xAA270F644A65D473, 0x5327AF7D84238CD0},
		},
		fp2elt{
			fpelt{0x5E06003D73C43EB1, 0x3EF69A49CB7E0237},
			fpelt{0x4E752648AC2EF0AB, 0x293EB1E26DD23B4E},
		},
	}
	if _R1toAffine(A) != A1000 {
		t.Fatalf("failed repeated add test")
	}
}

func testMul(mul mulfn, table []r2) bool {
	TEST_LOOPS := 1000

	curr := scalar{0x3AD457AB55456230, 0x3A8B3C2C6FD86E0C, 0x7E38F7C9CFBB9166, 0x0028FD6CBDA458F0}
	coeff := make([]scalar, TEST_LOOPS)
	for i := range coeff {
		curr[1] = curr[2]
		curr[2] += curr[0]
		coeff[i] = curr
	}

	A := _AffineToR1(affine{Gx, Gy})
	for i := range coeff {
		A = mul(coeff[i], A, table)
	}

	mulP := affine{
		fp2elt{
			fpelt{0xDFD2B477BD494BEF, 0x257C122BBFC94A1B},
			fpelt{0x769593547237C459, 0x469BF80CB5B11F01},
		},
		fp2elt{
			fpelt{0x281C5067996F3344, 0x0901B3817C0E936C},
			fpelt{0x4FE8C429915F1245, 0x570B948EACACE210},
		},
	}

	return _R1toAffine(A) == mulP
}

func TestMulWindowed(t *testing.T) {
	// Test multiplication by one and two
	A := _AffineToR1(affine{Gx, Gy})
	B := mulWindowed(toScalar(1), A, nil)
	if _R1toAffine(A) != _R1toAffine(B) {
		t.Fatalf("failed multiply-by-1 test (windowed)")
	}

	A2 := dbl(A)
	B2 := mulWindowed(toScalar(2), A, nil)
	if _R1toAffine(A2) != _R1toAffine(B2) {
		t.Fatalf("failed multiply-by-2 test (windowed)")
	}

	// Long-form multiplication test
	if !testMul(mulWindowed, nil) {
		t.Fatalf("failed large multiply test (windowed)")
	}

	// Test multiplication test with fixed base
	table := tableWindowed(A)
	B = mulWindowed(toScalar(1), A, table)
	if _R1toAffine(A) != _R1toAffine(B) {
		t.Fatalf("failed multiply-by-1 test (windowed, fixed base)")
	}

	A2 = dbl(A)
	B2 = mulWindowed(toScalar(2), A, table)
	if _R1toAffine(A2) != _R1toAffine(B2) {
		t.Fatalf("failed multiply-by-2 test (windowed, fixed base)")
	}
}

func TestDH(t *testing.T) {
	TEST_LOOPS := 100

	dhTest := func(label string, dh func(m scalar, P affine, table []r2) affine) {
		// Test that DH(m, P) == [392*m]P
		P := affine{Gx, Gy}
		failed := 0
		for i := 0; i < TEST_LOOPS; i += 1 {
			m := randScalar()
			Q1 := dh(m, P, nil)
			Q392 := mulWindowed(toScalar(392), _AffineToR1(P), nil)
			Q2 := _R1toAffine(mulWindowed(m, Q392, nil))
			if Q1 != Q2 {
				failed += 1
			}
			P = Q1
		}
		if failed > 0 {
			t.Fatalf("failed DH 392*m test (%s)", label)
		}

		// Test that DH has the symmetry property
		G := affine{Gx, Gy}
		failed = 0
		for i := 0; i < TEST_LOOPS; i += 1 {
			a := randScalar()
			b := randScalar()
			abG := dh(a, dh(b, G, nil), nil)
			baG := dh(b, dh(a, G, nil), nil)
			if abG != baG {
				t.Fatalf("failed DH symmetry test (%s)", label)
			}
		}
	}

	dhTest("windowed", dhWindowed)
}
