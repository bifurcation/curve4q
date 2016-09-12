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
	r4pt := r4{x, y, z}
	r1pt2 := r1{x2, y2, z2, t2, fp2One}

	if _R1toR2(r1pt) != r2pt {
		t.Fatalf("failed R1toR2")
	}

	if _R1toR3(r1pt) != r3pt {
		t.Fatalf("failed R1toR3")
	}

	if _R1toR4(r1pt) != r4pt {
		t.Fatalf("failed R1toR4 %v", _R1toR4(r1pt))
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

func TestEndo(t *testing.T) {
	TEST_LOOPS := 1000

	P := _AffineToR1(affine{Gx, Gy})
	for i := 0; i < TEST_LOOPS; i += 1 {
		P = phi(P)
	}
	phiP := affine{
		fp2elt{
			fpelt{0xD5B5A3061287DB16, 0x5550AAB9E7A620EE},
			fpelt{0xEC321E6CF33610FC, 0x3E61EBB9A1CB0210},
		},
		fp2elt{
			fpelt{0x7E2851D5A8E83FB9, 0x5474BF8EC55603AE},
			fpelt{0xA5077613491788D5, 0x5476093DBF8BF6BF},
		},
	}
	if _R1toAffine(P) != phiP {
		t.Fatalf("failed phi test")
	}

	P = _AffineToR1(affine{Gx, Gy})
	for i := 0; i < TEST_LOOPS; i += 1 {
		P = psi(P)
	}
	psiP := affine{
		fp2elt{
			fpelt{0xD8F3C8C24A2BC7E2, 0x75AF54EDB41A2B93},
			fpelt{0x4DE2466701F009A9, 0x065249F9EDE0C798},
		},
		fp2elt{
			fpelt{0x1C6E119ADD608104, 0x06DBB85BFFB7C21E},
			fpelt{0xFD234D6C4CFA3EC1, 0x060A30903424BF13},
		},
	}
	if _R1toAffine(P) != psiP {
		t.Fatalf("failed psi test")
	}
}

func TestRecode(t *testing.T) {
	// TODO TEST_LOOPS := 1000

	// Test decomposition
	testCases := []struct {
		in  scalar
		dec scalar
		m   []uint64
		d   []uint64
	}{
		{
			in:  scalar{0xfed8c8822ad9f1a7, 0x47b3e28c55984d43, 0x052d112f54981117, 0x92990788d66bf558},
			dec: scalar{0xa8ea3f673f711e51, 0xa08d1eae0b9e071d, 0x55c8df690050276f, 0x6396739dda88830f},
			m: []uint64{0, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 1, 0, 0, 0, 1,
				0, 0, 0, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 0, 0, 1,
				1, 1, 0, 0, 1, 1, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0,
				1, 0, 1, 0, 1, 1, 1, 0, 0, 0, 1, 0, 1, 0, 1, 0, 1},
			d: []uint64{7, 1, 0, 0, 6, 5, 0, 2, 5, 5, 1, 2, 0, 2, 2, 6,
				0, 1, 0, 4, 2, 1, 2, 7, 1, 5, 0, 5, 4, 0, 4, 0,
				2, 5, 5, 2, 3, 7, 2, 7, 6, 7, 3, 3, 7, 4, 2, 4,
				7, 4, 1, 7, 3, 4, 2, 7, 1, 3, 5, 2, 0, 7, 1, 7, 7},
		},
		{
			in:  scalar{0xe5230df341623348, 0x4b0c61edc8d1305f, 0xa214b88481393502, 0x48e5ca2a675ab49c},
			dec: scalar{0xa53ec4631945b875, 0x521c0ba1261c1934, 0x5c50ce912909185c, 0x93b3c70960b44bad},
			m: []uint64{0, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 1,
				0, 1, 0, 0, 0, 1, 0, 1, 0, 0, 1, 1, 0, 0, 0, 1,
				1, 0, 0, 0, 1, 1, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0,
				1, 1, 1, 1, 1, 0, 0, 1, 0, 1, 0, 0, 1, 0, 1, 0, 1},
			d: []uint64{4, 4, 7, 1, 5, 7, 2, 6, 3, 3, 7, 7, 3, 0, 4, 0,
				2, 2, 5, 6, 2, 3, 4, 0, 6, 7, 6, 3, 0, 7, 3, 7,
				7, 0, 0, 4, 6, 1, 0, 3, 6, 0, 1, 4, 7, 7, 6, 6,
				2, 0, 5, 1, 7, 4, 6, 2, 0, 1, 6, 4, 1, 6, 5, 6, 6},
		},
		{
			in:  scalar{0xe58a15c4b0f1c5e9, 0x05a1e8f7f6394d9b, 0xe4d9f3d5a5edfed3, 0xae20e251c36cfa5b},
			dec: scalar{0xa621ada9b3499c9f, 0x7cd17e0095e7aae6, 0x6e8d23b5bd10bb43, 0x7f18c69f3025234c},
			m: []uint64{1, 1, 1, 1, 0, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1, 1,
				0, 0, 1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 1,
				0, 0, 1, 0, 1, 0, 1, 1, 0, 1, 1, 0, 1, 0, 1, 1,
				0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 0, 0, 1, 0, 1, 0, 1},
			d: []uint64{2, 3, 5, 4, 0, 1, 6, 0, 7, 0, 7, 3, 2, 5, 7, 3,
				5, 4, 0, 5, 7, 2, 4, 4, 2, 1, 2, 1, 5, 4, 6, 3,
				6, 2, 0, 2, 0, 4, 6, 6, 2, 5, 7, 1, 0, 2, 6, 5,
				3, 3, 1, 5, 2, 5, 4, 6, 3, 2, 3, 0, 2, 2, 0, 7, 7},
		},
		{
			in:  scalar{0x34ff616d9806e10c, 0xb7e95036fd191ea1, 0x2cc00f1e3ac38f81, 0xb2c950abc87a5544},
			dec: scalar{0x9b30a872ebea83af, 0x8f6c73350447c9c3, 0x72fdc76e3456d087, 0x6ba39ba159b0c13d},
			m: []uint64{1, 1, 1, 0, 1, 0, 1, 1, 1, 0, 0, 0, 0, 0, 1, 0,
				1, 0, 1, 0, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 0,
				1, 0, 0, 1, 1, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 0,
				0, 0, 0, 1, 1, 0, 0, 1, 1, 0, 1, 1, 0, 0, 1, 0, 1},
			d: []uint64{7, 3, 6, 4, 0, 0, 5, 3, 5, 0, 0, 1, 3, 3, 4, 4,
				6, 2, 0, 3, 5, 6, 3, 4, 4, 0, 3, 4, 2, 6, 4, 0,
				5, 2, 1, 1, 3, 7, 2, 6, 1, 0, 5, 6, 3, 5, 6, 7,
				1, 3, 4, 4, 1, 5, 4, 1, 3, 3, 6, 4, 3, 5, 1, 7, 7},
		},
		{
			in:  scalar{0xdd4f4ce2b5f6b2de, 0x61f21fcebd67889f, 0x62340e9797788e00, 0x8e2958a1475ed707},
			dec: scalar{0xbe8f3583a0934333, 0xab45bf6d1bf80b37, 0x4a19fc5cffe97809, 0x5ea3baf1a1206442},
			m: []uint64{1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 1, 0, 1,
				1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 1, 1,
				1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 0, 1, 1, 0, 0, 1,
				1, 1, 1, 0, 0, 0, 1, 0, 1, 1, 1, 1, 1, 0, 1, 0, 1},
			d: []uint64{3, 5, 4, 7, 1, 1, 5, 5, 1, 1, 5, 6, 5, 1, 0, 6,
				2, 0, 0, 3, 1, 6, 4, 0, 4, 4, 5, 4, 4, 5, 5, 4,
				7, 0, 3, 0, 5, 2, 0, 3, 5, 0, 6, 0, 0, 0, 5, 0,
				0, 3, 5, 2, 0, 6, 7, 4, 5, 7, 4, 7, 4, 1, 7, 1, 1},
		},
	}

	uintSliceEq := func(a, b []uint64) bool {
		if len(a) != len(b) {
			return false
		}

		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	for _, test := range testCases {
		dec := decompose(test.in)
		if dec != test.dec {
			t.Fatalf("failed decompose test")
		}

		m, d := recode(dec)
		if !uintSliceEq(m, test.m) {
			t.Fatalf("failed recode test (sign masks) %v %v", m, test.m)
		}
		if !uintSliceEq(d, test.d) {
			t.Fatalf("failed recode test (digits) %v %v", d, test.d)
		}
	}
}

func TestMulEndo(t *testing.T) {
	// Test multiplication by one and two
	A := _AffineToR1(affine{Gx, Gy})
	B := mulEndo(toScalar(1), A, nil)
	if _R1toAffine(A) != _R1toAffine(B) {
		t.Fatalf("failed multiply-by-1 test (endo)")
	}

	A2 := dbl(A)
	B2 := mulEndo(toScalar(2), A, nil)
	if _R1toAffine(A2) != _R1toAffine(B2) {
		t.Fatalf("failed multiply-by-2 test (endo)")
	}

	// Long-form multiplication test
	if !testMul(mulEndo, nil) {
		t.Fatalf("failed large multiply test (endo)")
	}

	// Test multiplication test with fixed base
	table := tableEndo(A)
	B = mulEndo(toScalar(1), A, table)
	if _R1toAffine(A) != _R1toAffine(B) {
		t.Fatalf("failed multiply-by-1 test (endo, fixed base)")
	}

	A2 = dbl(A)
	B2 = mulEndo(toScalar(2), A, table)
	if _R1toAffine(A2) != _R1toAffine(B2) {
		t.Fatalf("failed multiply-by-2 test (endo, fixed base)")
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
	dhTest("endo", dhEndo)
}
