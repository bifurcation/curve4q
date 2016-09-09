package curve4q

import ()

var (
	// p = 2^127 - 1
	p0 uint64 = 0xffffffffffffffff
	p1 uint64 = 0x7fffffffffffffff

	_m  uint64 = 0xffffffffffffffff
	_W2 uint64 = 32                 // Half word size in bits
	_M2 uint64 = 0x00000000ffffffff // Half word mask
)

type fpelt [2]uint64
type fp2elt [2]fpelt

var (
	p      = fpelt{p0, p1}
	fpZero = fpelt{0, 0}
	fpOne  = fpelt{1, 0}
	fpTwo  = fpelt{2, 0}
	fpHalf = fpelt{0x0000000000000000, 0x4000000000000000}
	fp2One = fp2elt{fpOne, fpZero}
	fp2Two = fp2elt{fpTwo, fpZero}
)

func fpint(x uint64) fpelt {
	return fpelt{x, 0}
}

// Adapted from "math/big" internals
// https://golang.org/src/math/big/arith.go#L34
// z1<<_W + z0 = x+y+c, with c == 0 or 1
func wadd(x, y, c uint64) (z1, z0 uint64) {
	yc := y + c
	z0 = x + yc
	if z0 < x || yc < y {
		z1 = 1
	}
	return
}

// Adapted from "math/big" internals
// https://golang.org/src/math/big/arith.go#L54
// z1<<_W + z0 = x*y
func wmul(x, y uint64) (z1, z0 uint64) {
	x0 := x & _M2
	x1 := x >> _W2
	y0 := y & _M2
	y1 := y >> _W2
	w0 := x0 * y0
	t := x1*y0 + w0>>_W2
	w1 := t & _M2
	w2 := t >> _W2
	w1 += x0 * y1
	z1 = x1*y1 + w2 + w1>>_W2
	z0 = x * y
	return
}

// Constant time MSB calculation
// Adapted from OpenSSL constant_time_is_zero
func wzero(x uint64) uint64 {
	return 0 - ((^x & (x - 1)) >> 63)
}

// Constant-time select
// switch (c) {
//   case 1: return x
//   case 2: return y
// }
func fpselect(c uint64, x, y fpelt) fpelt {
	m := c * _m
	return fpelt{
		y[0] ^ (m & (x[0] ^ y[0])),
		y[1] ^ (m & (x[1] ^ y[1])),
	}
}

// To compute k mod p
//  int i = (k & p) + (k >> s);
//  return (i >= p) ? i - p : i;
func fpreduce(x *fpelt) {
	var c uint64
	s := x[1] >> 63
	x[1] &= p1
	c, x[0] = wadd(x[0], 0, s)
	_, x[1] = wadd(x[1], 0, c)

	gt := ^wzero(x[1] >> 63)
	eq := wzero((x[0] ^ p0) | (x[1] ^ p1))
	m := (gt | eq) & _m

	x[0] ^= m
	x[1] ^= m
	x[1] &= p1
	return
}

// (x << 64) % p
func fpreduce64(x *fpelt) {
	x1 := (x[0] & p1) + (x[1] >> 63)
	x0 := (x[1] << 1) + (x[0] >> 63)
	m := (x1 >> 63) * _m
	x[0] = x0 ^ m
	x[1] = x1 ^ m
	return
}

// (x << 128) % p
func fpreduce128(x *fpelt) {
	x1 := (x[1] << 1) + (x[0] >> 63)
	x0 := x[0] << 1
	m := (x1 >> 63) * _m
	x[0] = x0 ^ m
	x[1] = x1 ^ m
	return
}

// Don't need to worry about overflow as long as both x and y
// are reduced; the extra bit at the top holds the carry, then
// gets used in the reduction
func fpadd(x, y fpelt) (z fpelt) {
	var c uint64
	c, z[0] = wadd(x[0], y[0], 0)
	_, z[1] = wadd(x[1], y[1], c)
	fpreduce(&z)
	return
}

func fpneg(x fpelt) (z fpelt) {
	z[0] = x[0] ^ p0
	z[1] = x[1] ^ p1
	fpreduce(&z)
	return
}

func fpsub(x, y fpelt) fpelt {
	return fpadd(x, fpneg(y))
}

func fpmul(x, y fpelt) (z fpelt) {
	// x * y = (D << 128) + (C << 64) + (B << 64) + A
	A1, A0 := wmul(x[0], y[0])
	B1, B0 := wmul(x[0], y[1])
	C1, C0 := wmul(x[1], y[0])
	D1, D0 := wmul(x[1], y[1])

	A := fpelt{A0, A1}
	B := fpelt{B0, B1}
	C := fpelt{C0, C1}
	D := fpelt{D0, D1}

	fpreduce(&A)
	fpreduce64(&B)
	fpreduce64(&C)
	fpreduce128(&D)

	z = A
	z = fpadd(z, B)
	z = fpadd(z, C)
	z = fpadd(z, D)
	return
}

func fpinv(x fpelt) fpelt {
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

func fpinvsqrt(x fpelt) fpelt {
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

func fpsqr(x fpelt) (z fpelt) {
	A1, A0 := wmul(x[0], x[0])
	B1, B0 := wmul(x[0], x[1])
	C1, C0 := wmul(x[1], x[1])

	A := fpelt{A0, A1}
	B := fpelt{(B0 << 1), (B1 << 1) + (B0 >> 63)}
	C := fpelt{C0, C1}

	fpreduce(&A)
	fpreduce64(&B)
	fpreduce128(&C)

	z = A
	z = fpadd(z, B)
	z = fpadd(z, C)
	return
}

func fp2select(c uint64, x, y fp2elt) fp2elt {
	return fp2elt{fpselect(c, x[0], y[0]), fpselect(c, x[1], y[1])}
}

func fp2add(x, y fp2elt) (z fp2elt) {
	return fp2elt{fpadd(x[0], y[0]), fpadd(x[1], y[1])}
}

func fp2sub(x, y fp2elt) (z fp2elt) {
	return fp2elt{fpsub(x[0], y[0]), fpsub(x[1], y[1])}
}

func fp2neg(x fp2elt) fp2elt {
	return fp2elt{fpneg(x[0]), fpneg(x[1])}
}

func fp2conj(x fp2elt) (z fp2elt) {
	return fp2elt{x[0], fpneg(x[1])}
}

func fp2mul(x, y fp2elt) (z fp2elt) {
	t00 := fpmul(x[0], y[0])
	t01 := fpmul(x[0], y[1])
	t10 := fpmul(x[1], y[0])
	t11 := fpmul(x[1], y[1])

	z[0] = fpsub(t00, t11)
	z[1] = fpadd(t01, t10)
	return
}

func fp2sqr(x fp2elt) (z fp2elt) {
	t00 := fpsqr(x[0])
	t11 := fpsqr(x[1])
	t01 := fpmul(x[0], x[1])

	z[0] = fpsub(t00, t11)
	z[1] = fpadd(t01, t01)
	return
}

func fp2inv(x fp2elt) (z fp2elt) {
	invmag := fpinv(fpadd(fpsqr(x[0]), fpsqr(x[1])))
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
	if fpmul(c, s) != fpOne {
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
