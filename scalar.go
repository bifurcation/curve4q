package curve4q

type scalar [4]uint64

var (
	N = scalar{0x2fb2540ec7768ce7, 0xdfbd004dfe0f7999, 0xf05397829cbc14e5, 0x0029cbc14e5e0a72}
)

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
