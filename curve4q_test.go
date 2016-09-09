package curve4q

import (
	"bytes"
	"encoding/hex"
	"testing"
)

/********** Utilities **********/

var (
	G1 = r1{Gx, Gy, fp2One, Gx, Gy}
	O1 = r1{Ox, Oy, fp2One, Ox, Oy}
)

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

	encTest := encode(Gx, Gy)
	if !bytes.Equal(encTest, GEnc) {
		t.Fatalf("Encode test failed %x %x", encTest, GEnc)
	}

	decX, decY := decode(GEnc)
	if decX != Gx || decY != Gy {
		t.Fatalf("Decode test failed %x %x", encTest, GEnc)
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
	td2 := fp2mul(fp2Two, fp2mul(d, tt))

	r1pt := r1{x, y, z, ta, tb}
	r2pt := r2{xy, yx, z2, td2}
	r3pt := r3{xy, yx, z, tt}
	r4pt := r4{x2, y2, z2}

	if _R1toR2(r1pt) != r2pt {
		t.Fatalf("failed R1toR2")
	}

	if _R1toR3(r1pt) != r3pt {
		t.Fatalf("failed R1toR3")
	}

	if _R2toR4(r2pt) != r4pt {
		t.Fatalf("failed R2toR4")
	}
}

func TestCore(t *testing.T) {
	// Test doubling
	A := _affineToR1(affine{Gx, Gy})
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
	G := _affineToR1(affine{Gx, Gy})
	O := _affineToR1(affine{Ox, Oy})
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
