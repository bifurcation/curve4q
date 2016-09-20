package curve4q

type scalar [4]uint64

var (
	N = scalar{0x2fb2540ec7768ce7, 0xdfbd004dfe0f7999, 0xf05397829cbc14e5, 0x0029cbc14e5e0a72}

	_m  uint64 = 0xffffffffffffffff
	_W2 uint64 = 32                 // Half word size in bits
	_M2 uint64 = 0x00000000ffffffff // Half word mask
)

func wselect(c, x1, x0 uint64) uint64 {
	return x0 ^ ((c * _m) & (x1 ^ x0))
}

// XXX: NOT CONSTANT TIME
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

// XXX: NOT CONSTANT TIME
// Adapted from "math/big" internals
// https://golang.org/src/math/big/arith.go#L44
// z1<<_W + z0 = x-y-c, with c == 0 or 1
func wsub(x, y, c uint64) (z1, z0 uint64) {
	yc := y + c
	z0 = x - yc
	if z0 > x || yc < y {
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

func abs(x int) int {
	x64 := uint64(x)
	x64n := uint64(-1 * x)
	return int(wselect(x64>>63, x64n, x64))
}

func sadd(x scalar, y scalar) (z scalar) {
	var c uint64
	c, z[0] = wadd(x[0], y[0], 0)
	c, z[1] = wadd(x[1], y[1], c)
	c, z[2] = wadd(x[2], y[2], c)
	_, z[3] = wadd(x[3], y[3], c) // XXX overflow?
	return
}

func ssub(x scalar, y scalar) (z scalar) {
	var c uint64
	c, z[0] = wsub(x[0], y[0], 0)
	c, z[1] = wsub(x[1], y[1], c)
	c, z[2] = wsub(x[2], y[2], c)
	_, z[3] = wsub(x[3], y[3], c) // XXX underflow?
	return
}

func ssubi(x scalar, y int) (z scalar) {
	y64 := uint64(y)
	y64n := uint64(-1 * y)
	neg := (y64 >> 63)
	a := wselect(neg, y64n, y64)
	xp := sadd(x, scalar{a, 0, 0, 0})
	xm := ssub(x, scalar{a, 0, 0, 0})
	return sselect(neg, xp, xm)
}

func srsh4(x scalar) (z scalar) {
	z[3] = x[3] >> 4
	z[2] = (x[2] >> 4) | ((x[3] & 0x0f) << 60)
	z[1] = (x[1] >> 4) | ((x[2] & 0x0f) << 60)
	z[0] = (x[0] >> 4) | ((x[1] & 0x0f) << 60)
	return
}

func smulw(x scalar, y uint64) (c uint64, z scalar) {
	A1, A0 := wmul(x[0], y)
	B1, B0 := wmul(x[1], y)
	C1, C0 := wmul(x[2], y)
	D1, D0 := wmul(x[3], y)

	z[0] = A0
	c, z[1] = wadd(A1, B0, 0)
	c, z[2] = wadd(B1, C0, c)
	c, z[3] = wadd(C1, D0, c)
	c = c + D1
	return
}

// Note: Could possibly be made faster, as in FourQlib
func smultrunc(x scalar, y scalar) uint64 {
	var c, hi uint64
	var lo scalar

	hi, lo = smulw(x, y[0])

	_, as := smulw(x, y[1])
	c, lo[1] = wadd(lo[1], as[0], 0)
	c, lo[2] = wadd(lo[2], as[1], c)
	c, lo[3] = wadd(lo[3], as[2], c)
	_, hi = wadd(hi, as[3], c)

	_, as = smulw(x, y[2])
	c, lo[2] = wadd(lo[2], as[0], 0)
	c, lo[3] = wadd(lo[3], as[1], c)
	_, hi = wadd(hi, as[2], c)

	_, as = smulw(x, y[3])
	c, lo[3] = wadd(lo[3], as[0], 0)
	_, hi = wadd(hi, as[1], c)

	return hi
}

// Note: Super-basic implementation
// Note: Not constant-time
func smodN(x scalar) (y scalar) {
	y = x
	for y[3] >= N[3] {
		y = ssub(y, N)
	}
	return
}

func sselect(c uint64, x1 scalar, x0 scalar) (y scalar) {
	m := c * _m
	y[0] = x0[0] ^ (m & (x1[0] ^ x0[0]))
	y[1] = x0[1] ^ (m & (x1[1] ^ x0[1]))
	y[2] = x0[2] ^ (m & (x1[2] ^ x0[2]))
	y[3] = x0[3] ^ (m & (x1[3] ^ x0[3]))
	return
}
