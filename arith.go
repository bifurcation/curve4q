package curve4q

import ()

var (
	// p = 2^127 - 1
	p0 uint64 = 0xffffffffffffffff
	p1 uint64 = 0x7fffffffffffffff
)

var (
	fp2M = 0 // Multiplies
	fp2S = 0 // Squares
	fp2A = 0 // Adds
	fp2I = 0 // Inverses
)

func clearCounters() {
	fp2M = 0
	fp2S = 0
	fp2A = 0
	fp2I = 0
}

type fpelt [2]uint64  // Compact representation of base field elements
type fpselt [5]int32  // "Spread" representation of base field elements
type fp2elt [2]fpselt // Elements of the quadratic extension field

func (x fpelt) eq(y fpelt) bool {
	return x == y
}

func (x fpselt) eq(y fpselt) bool {
	return unspread(x) == unspread(y)
}

func (x fp2elt) eq(y fp2elt) bool {
	return x[0].eq(y[0]) && x[1].eq(y[1])
}

var (
	p      = fpelt{p0, p1}
	fpZero = fpselt{0, 0, 0, 0, 0}
	fpOne  = fpselt{1, 0, 0, 0, 0}
	fpTwo  = fpselt{2, 0, 0, 0, 0}
	fpHalf = spread(fpelt{0x0000000000000000, 0x4000000000000000})
	fp2One = fp2elt{fpOne, fpZero}
	fp2Two = fp2elt{fpTwo, fpZero}
)

func fpint(x uint64) fpselt {
	return spread(fpelt{x, 0})
}

// To compute k mod p
//   i = (k & p) + (k >> s);
//   return (i >= p) ? i - p : i;
func fpreduce(x *fpelt) {
	var c uint64
	s := x[1] >> 63
	x[1] &= p1
	c, x[0] = wadd(x[0], 0, s)
	_, x[1] = wadd(x[1], 0, c)

	gt := ^is_zero(x[1] >> 63)
	eq := is_zero((x[0] ^ p0) | (x[1] ^ p1))
	m := (gt | eq) & _m

	x[0] ^= m
	x[1] ^= m
	x[1] &= p1
	return
}

// Borrowed from golang.org/x/crypto/curve25519
// load3 reads a 24-bit, little-endian value from in.
func load3(in []byte) int64 {
	var r int64
	r = int64(in[0])
	r |= int64(in[1]) << 8
	r |= int64(in[2]) << 16
	return r
}

// Borrowed from golang.org/x/crypto/curve25519
// load4 reads a 32-bit, little-endian value from in.
func load4(in []byte) int64 {
	var r int64
	r = int64(in[0])
	r |= int64(in[1]) << 8
	r |= int64(in[2]) << 16
	r |= int64(in[3]) << 24
	return r
}

func spread(x fpelt) (z fpselt) {
	src := []byte{
		byte(x[0] >> (8 * 0)), byte(x[0] >> (8 * 1)), byte(x[0] >> (8 * 2)), byte(x[0] >> (8 * 3)),
		byte(x[0] >> (8 * 4)), byte(x[0] >> (8 * 5)), byte(x[0] >> (8 * 6)), byte(x[0] >> (8 * 7)),
		byte(x[1] >> (8 * 0)), byte(x[1] >> (8 * 1)), byte(x[1] >> (8 * 2)), byte(x[1] >> (8 * 3)),
		byte(x[1] >> (8 * 4)), byte(x[1] >> (8 * 5)), byte(x[1] >> (8 * 6)), byte(x[1] >> (8 * 7)),
	}

	h0 := load4(src)           // 32 bits = 26 + 6
	h1 := load3(src[4:]) << 6  // 30 bits = 26 + 4
	h2 := load3(src[7:]) << 4  // 28 bits = 26 + 2
	h3 := load3(src[10:]) << 2 // 26 bits = 26
	h4 := load3(src[13:])      // 24 bits

	carry := (h4 + (1 << 22)) >> 23
	h0 += carry
	h4 -= carry << 23

	carry = (h0 + (1 << 25)) >> 26
	h1 += carry
	h0 -= carry << 26

	carry = (h1 + (1 << 25)) >> 26
	h2 += carry
	h1 -= carry << 26

	carry = (h2 + (1 << 25)) >> 26
	h3 += carry
	h2 -= carry << 26

	carry = (h3 + (1 << 25)) >> 26
	h4 += carry
	h3 -= carry << 26

	z[0] = int32(h0)
	z[1] = int32(h1)
	z[2] = int32(h2)
	z[3] = int32(h3)
	z[4] = int32(h4)
	return
}

func unspread(x fpselt) (z fpelt) {
	q := (x[4] + (1 << 22)) >> 23
	q = (x[0] + q) >> 26
	q = (x[1] + q) >> 26
	q = (x[2] + q) >> 26
	q = (x[3] + q) >> 26
	q = (x[4] + q) >> 26

	x[0] += q

	carry := x[0] >> 26
	x[1] += carry
	x[0] -= carry << 26

	carry = x[1] >> 26
	x[2] += carry
	x[1] -= carry << 26

	carry = x[2] >> 26
	x[3] += carry
	x[2] -= carry << 26

	carry = x[3] >> 26
	x[4] += carry
	x[3] -= carry << 26

	carry = x[4] >> 23
	x[4] -= carry << 23

	z[0] = uint64(x[0]) | (uint64(x[1]) << 26) | (uint64(x[2]) << 52)
	z[1] = (uint64(x[2]) >> 12) | (uint64(x[3]) << 14) | (uint64(x[4]) << 40)
	return
}

func fpadd(x, y fpselt) (z fpselt) {
	z[0] = x[0] + y[0]
	z[1] = x[1] + y[1]
	z[2] = x[2] + y[2]
	z[3] = x[3] + y[3]
	z[4] = x[4] + y[4]
	return
}

func fpsub(x, y fpselt) (z fpselt) {
	z[0] = x[0] - y[0]
	z[1] = x[1] - y[1]
	z[2] = x[2] - y[2]
	z[3] = x[3] - y[3]
	z[4] = x[4] - y[4]
	return
}

func fpneg(x fpselt) (z fpselt) {
	z[0] = -x[0]
	z[1] = -x[1]
	z[2] = -x[2]
	z[3] = -x[3]
	z[4] = -x[4]
	return
}

func fpmul(x, y fpselt) (z fpselt) {
	x0 := int64(x[0])
	x1 := int64(x[1])
	x2 := int64(x[2])
	x3 := int64(x[3])
	x4 := int64(x[4])

	y0 := int64(y[0])
	y1 := int64(y[1])
	y2 := int64(y[2])
	y3 := int64(y[3])
	y4 := int64(y[4])

	x0y0 := x0 * y0
	x0y1 := x0 * y1
	x0y2 := x0 * y2
	x0y3 := x0 * y3
	x0y4 := x0 * y4
	x1y0 := x1 * y0
	x1y1 := x1 * y1
	x1y2 := x1 * y2
	x1y3 := x1 * y3
	x1y4 := x1 * y4
	x2y0 := x2 * y0
	x2y1 := x2 * y1
	x2y2 := x2 * y2
	x2y3 := x2 * y3
	x2y4 := x2 * y4
	x3y0 := x3 * y0
	x3y1 := x3 * y1
	x3y2 := x3 * y2
	x3y3 := x3 * y3
	x3y4 := x3 * y4
	x4y0 := x4 * y0
	x4y1 := x4 * y1
	x4y2 := x4 * y2
	x4y3 := x4 * y3
	x4y4 := x4 * y4

	h0 := x0y0 + ((x1y4 + x2y3 + x3y2 + x4y1) << 3)
	h1 := x0y1 + x1y0 + ((x2y4 + x3y3 + x4y2) << 3)
	h2 := x0y2 + x1y1 + x2y0 + ((x3y4 + x4y3) << 3)
	h3 := x0y3 + x1y2 + x2y1 + x3y0 + (x4y4 << 3)
	h4 := x0y4 + x1y3 + x2y2 + x3y1 + x4y0

	carry := (h0 + (1 << 25)) >> 26
	h1 += carry
	h0 -= carry << 26

	carry = (h1 + (1 << 25)) >> 26
	h2 += carry
	h1 -= carry << 26

	carry = (h2 + (1 << 25)) >> 26
	h3 += carry
	h2 -= carry << 26

	carry = (h3 + (1 << 25)) >> 26
	h4 += carry
	h3 -= carry << 26

	carry = (h4 + (1 << 22)) >> 23
	h0 += carry
	h4 -= carry << 23

	z[0] = int32(h0)
	z[1] = int32(h1)
	z[2] = int32(h2)
	z[3] = int32(h3)
	z[4] = int32(h4)
	return
}

func fpsqr(x fpselt) (z fpselt) {
	x0 := int64(x[0])
	x1 := int64(x[1])
	x2 := int64(x[2])
	x3 := int64(x[3])
	x4 := int64(x[4])

	x1_2 := x1 + x1
	x2_2 := x2 + x2
	x3_2 := x3 + x3
	x4_2 := x4 + x4

	x0x0 := x0 * x0
	x1x1 := x1 * x1
	x2x2 := x2 * x2
	x3x3 := x3 * x3
	x4x4 := x4 * x4

	x0x1_2 := x0 * x1_2
	x0x2_2 := x0 * x2_2
	x0x3_2 := x0 * x3_2
	x0x4_2 := x0 * x4_2
	x1x2_2 := x1 * x2_2
	x1x3_2 := x1 * x3_2
	x1x4_2 := x1 * x4_2
	x2x3_2 := x2 * x3_2
	x2x4_2 := x2 * x4_2
	x3x4_2 := x3 * x4_2

	h0 := x0x0 + ((x1x4_2 + x2x3_2) << 3)
	h1 := x0x1_2 + ((x2x4_2 + x3x3) << 3)
	h2 := x0x2_2 + x1x1 + (x3x4_2 << 3)
	h3 := x0x3_2 + x1x2_2 + (x4x4 << 3)
	h4 := x0x4_2 + x1x3_2 + x2x2

	carry := (h0 + (1 << 25)) >> 26
	h1 += carry
	h0 -= carry << 26

	carry = (h1 + (1 << 25)) >> 26
	h2 += carry
	h1 -= carry << 26

	carry = (h2 + (1 << 25)) >> 26
	h3 += carry
	h2 -= carry << 26

	carry = (h3 + (1 << 25)) >> 26
	h4 += carry
	h3 -= carry << 26

	carry = (h4 + (1 << 22)) >> 23
	h0 += carry
	h4 -= carry << 23

	z[0] = int32(h0)
	z[1] = int32(h1)
	z[2] = int32(h2)
	z[3] = int32(h3)
	z[4] = int32(h4)
	return
}

func fpinv(x fpselt) fpselt {
	t2 := fpsqr(x)              // 2
	t2 = fpmul(x, t2)           // 3
	t3 := fpsqr(t2)             // 6
	t3 = fpsqr(t3)              // 12
	t3 = fpmul(t2, t3)          // 15
	t4 := fpsqr(t3)             // 30
	t4 = fpsqr(t4)              // 60
	t4 = fpsqr(t4)              // 120
	t4 = fpsqr(t4)              // 240
	t4 = fpmul(t3, t4)          // 2^8 - 2^0
	t5 := fpsqr(t4)             // 2^9 - 2^1
	for i := 0; i < 7; i += 1 { // 2^16 - 2^8
		t5 = fpsqr(t5)
	}
	t5 = fpmul(t4, t5)           // 2^16 - 2^0
	t2 = fpsqr(t5)               // 2^17 - 2^1
	for i := 0; i < 15; i += 1 { // 2^32 - 2^16
		t2 = fpsqr(t2)
	}
	t2 = fpmul(t5, t2)           // 2^32 - 2^0
	t1 := fpsqr(t2)              // 2^33 - 2^1
	for i := 0; i < 31; i += 1 { // 2^64 - 2^32
		t1 = fpsqr(t1)
	}
	t1 = fpmul(t1, t2)           // 2^64 - 2^0
	for i := 0; i < 32; i += 1 { // 2^96 - 2^32
		t1 = fpsqr(t1)
	}
	t1 = fpmul(t1, t2)           // 2^96 - 2^0
	for i := 0; i < 16; i += 1 { // 2^112 - 2^16
		t1 = fpsqr(t1)
	}
	t1 = fpmul(t1, t5)          // 2^112 - 2^0
	for i := 0; i < 8; i += 1 { // 2^120 - 2^8
		t1 = fpsqr(t1)
	}
	t1 = fpmul(t1, t4)          // 2^120 - 2^0
	for i := 0; i < 4; i += 1 { // 2^124 - 2^4
		t1 = fpsqr(t1)
	}
	t1 = fpmul(t1, t3)  // 2^124 - 2^0
	t1 = fpsqr(t1)      // 2^125 - 2^1
	t1 = fpmul(t1, x)   // 2^125 - 2^0
	t1 = fpsqr(t1)      // 2^126 - 2^1
	t1 = fpsqr(t1)      // 2^127 - 2^2
	return fpmul(t1, x) // 2^127 - 3
}

func fpinvsqrt(x fpselt) fpselt {
	xp := fpmul(x, x)             // 2
	xp = fpmul(xp, xp)            // 4
	xp = fpmul(xp, x)             // 5
	xp = fpmul(fpmul(xp, xp), xp) // 15
	xp = fpmul(fpmul(xp, xp), x)  // 31 = 2^5 - 1

	accum := xp
	for i := 0; i < 24; i += 1 {
		for i := 0; i < 5; i += 1 {
			xp = fpsqr(xp) // 2^(5(i+1)) - 2^(5i)
		}
		accum = fpmul(xp, accum) // 2^(5(i+1)) - 1
	}
	return accum
}

func fpselect(c int32, x1, x0 fpselt) (z fpselt) {
	m := -c
	z[0] = x0[0] ^ (m & (x1[0] ^ x0[0]))
	z[1] = x0[1] ^ (m & (x1[1] ^ x0[1]))
	z[2] = x0[2] ^ (m & (x1[2] ^ x0[2]))
	z[3] = x0[3] ^ (m & (x1[3] ^ x0[3]))
	z[4] = x0[4] ^ (m & (x1[4] ^ x0[4]))
	return
}

func fp2select(c int32, x, y fp2elt) fp2elt {
	return fp2elt{fpselect(c, x[0], y[0]), fpselect(c, x[1], y[1])}
}

func fp2add(x, y fp2elt) (z fp2elt) {
	fp2A += 1
	return fp2elt{fpadd(x[0], y[0]), fpadd(x[1], y[1])}
}

func fp2sub(x, y fp2elt) (z fp2elt) {
	fp2A += 1
	return fp2elt{fpsub(x[0], y[0]), fpsub(x[1], y[1])}
}

func fp2neg(x fp2elt) fp2elt {
	return fp2elt{fpneg(x[0]), fpneg(x[1])}
}

func fp2conj(x fp2elt) (z fp2elt) {
	return fp2elt{x[0], fpneg(x[1])}
}

func fp2mul(x, y fp2elt) (z fp2elt) {
	fp2M += 1
	t00 := fpmul(x[0], y[0])
	t11 := fpmul(x[1], y[1])

	xsum := fpadd(x[0], x[1])
	ysum := fpadd(y[0], y[1])
	tcross := fpmul(xsum, ysum)
	tres := fpsub(tcross, t00)

	z[0] = fpsub(t00, t11)
	z[1] = fpsub(tres, t11)
	return
}

func fp2sqr(x fp2elt) (z fp2elt) {
	fp2S += 1
	a2 := fpsqr(x[0])
	b2 := fpsqr(x[1])
	ab := fpmul(x[0], x[1])
	z[0] = fpsub(a2, b2)
	z[1] = fpadd(ab, ab)
	return
}

func fp2inv(x fp2elt) (z fp2elt) {
	fp2I += 1
	invmag := fpinv(fpadd(fpsqr(x[0]), fpsqr(x[1])))
	fp2M -= 2
	return fp2elt{fpmul(invmag, x[0]), fpmul(invmag, fpneg(x[1]))}
}

func fp2invsqrt(x fp2elt) (z fp2elt) {
	if x[1] == fpZero {
		t := fpinvsqrt(x[0])
		c := fpmul(x[0], fpsqr(t))
		if c == fpOne {
			return fp2elt{t, fpZero}
		} else {
			return fp2elt{fpZero, t}
		}
	}

	n := fpadd(fpsqr(x[0]), fpsqr(x[1]))
	s := fpinvsqrt(n)
	c := fpmul(n, s)
	if !fpOne.eq(fpmul(c, s)) {
		// XXX: Should handle this more gracefully
		panic("Number is not square")
	}

	delta := fpmul(fpadd(x[0], c), fpHalf)
	g := fpinvsqrt(delta)
	h := fpmul(delta, g)
	if fpmul(h, g) != fpOne {
		delta = fpmul(fpsub(x[0], c), fpHalf)
		g = fpinvsqrt(delta)
		h = fpmul(delta, g)
	}

	return fp2elt{
		fpmul(h, s),
		fpneg(fpmul(fpmul(fpmul(x[1], s), g), fpHalf)),
	}
}
