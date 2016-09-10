package curve4q

import (
	"encoding/binary"
)

var (
	basePoint      = affine{Gx, Gy}
	basePoint392   = mulWindowed(scalar{392, 0, 0, 0}, _AffineToR1(basePoint), nil)
	basePointTable = tableWindowed(basePoint392)
)

func decodeScalar(in *[32]byte) (m scalar) {
	m[0] = binary.LittleEndian.Uint64(in[0:8])
	m[1] = binary.LittleEndian.Uint64(in[8:16])
	m[2] = binary.LittleEndian.Uint64(in[16:24])
	m[3] = binary.LittleEndian.Uint64(in[24:32]) & 0x000fffffffffffff
	return
}

func ScalarBaseMult(dst, in *[32]byte) {
	m := decodeScalar(in)
	Q := dhWindowed(m, basePoint, basePointTable)
	buf := encode(Q)
	copy(dst[:], buf)
}

func ScalarMult(dst, in, base *[32]byte) {
	m := decodeScalar(in)
	P := decode(base[:])
	Q := dhWindowed(m, P, nil)
	buf := encode(Q)
	copy(dst[:], buf)
}
