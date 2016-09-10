package curve4q

import (
	"encoding/binary"
)

/********** Definitions **********/

var (
	// Curve parameter as a field element tuple
	d = fp2elt{
		fpelt{0x0000000000000142, 0x00000000000000e4},
		fpelt{0xb3821488f1fc0c8d, 0x5e472f846657e0fc},
	}

	// Inverse of the curve parameter
	dinv = fp2inv(d)

	// Affine coordinates for the neutral point
	Ox = fp2elt{fpZero, fpZero}
	Oy = fp2elt{fpOne, fpZero}

	// Affine coordinates for the base point
	Gx = fp2elt{
		fpelt{0x286592AD7B3833AA, 0x1A3472237C2FB305},
		fpelt{0x96869FB360AC77F6, 0x1E1F553F2878AA9C},
	}
	Gy = fp2elt{
		fpelt{0xB924A2462BCBB287, 0x0E3FEE9BA120785A},
		fpelt{0x49A7C344844C8B5C, 0x6E1C4AF8630E0242},
	}
)

// E: -x^2 + y^2 = 1 + d * x^2 * y^2
func pointOnCurve(X, Y fp2elt) bool {
	X2 := fp2sqr(X)
	Y2 := fp2sqr(Y)
	LHS := fp2sub(Y2, X2)
	RHS := fp2add(fp2One, fp2mul(fp2mul(d, X2), Y2))
	return LHS == RHS
}

/********** Point encoding / decoding **********/

// "Sign" bit used in compression / decompression
// s = (x0 != 0)? (x0 >> 127) : (x1 >> 127)
func sign(x fp2elt) byte {
	x0z := wzero(x[0][0] | x[0][1])
	s0 := x[0][1] >> 63
	s1 := x[1][1] >> 63
	return byte((s1 ^ (x0z & (s0 ^ s1))) & 0xFF)
}

func encode(P affine) []byte {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf[0:8], P.Y[0][0])
	binary.LittleEndian.PutUint64(buf[8:16], P.Y[0][1])
	binary.LittleEndian.PutUint64(buf[16:24], P.Y[1][0])
	binary.LittleEndian.PutUint64(buf[24:32], P.Y[1][1])
	buf[31] |= sign(P.X)
	return buf
}

func decode(buf []byte) (P affine) {
	// XXX: Should handle these more gracefully
	if len(buf) != 32 {
		panic("Malformed point: length is not 32")
	}
	if buf[15]&0x80 != 0x00 {
		panic("Malformed point: reserved bit is not zero")
	}

	s := buf[31] >> 7
	buf[31] &= 0x7f

	y00 := binary.LittleEndian.Uint64(buf[0:8])
	y01 := binary.LittleEndian.Uint64(buf[8:16])
	y10 := binary.LittleEndian.Uint64(buf[16:24])
	y11 := binary.LittleEndian.Uint64(buf[24:32])
	P.Y = fp2elt{fpelt{y00, y01}, fpelt{y10, y11}}

	y2 := fp2sqr(P.Y)
	y21 := fp2sub(y2, fp2One)
	dy21 := fp2add(fp2mul(d, y2), fp2One)
	sqrt := fp2invsqrt(fp2mul(y21, dy21))
	P.X = fp2mul(y21, sqrt)

	if s != sign(P.X) {
		P.X = fp2neg(P.X)
	}

	// XXX: Handle more gracefully
	if !pointOnCurve(P.X, P.Y) {
		panic("Malformed point: not on curve")
	}

	return
}

/********** Alternative Point Representations and Addition Laws **********/

type affine struct{ X, Y fp2elt }
type r1 struct{ X, Y, Z, Ta, Tb fp2elt }
type r2 struct{ N, D, E, F fp2elt }
type r3 struct{ N, D, Z, T fp2elt }
type r4 struct{ X, Y, Z fp2elt }

func _AffineToR1(P affine) (Q r1) {
	Q = r1{P.X, P.Y, fp2One, P.X, P.Y}
	return
}

func _R1toAffine(P r1) (Q affine) {
	Zi := fp2inv(P.Z)
	Q = affine{fp2mul(P.X, Zi), fp2mul(P.Y, Zi)}
	return
}

func _R1toR2(P r1) (Q r2) {
	Q.N = fp2add(P.X, P.Y)
	Q.D = fp2sub(P.Y, P.X)
	Q.E = fp2add(P.Z, P.Z)
	Q.F = fp2mul(fp2mul(fp2Two, d), fp2mul(P.Ta, P.Tb))
	return
}

func _R1toR3(P r1) (Q r3) {
	Q.N = fp2add(P.X, P.Y)
	Q.D = fp2sub(P.Y, P.X)
	Q.Z = P.Z
	Q.T = fp2mul(P.Ta, P.Tb)
	return
}

// Note: We pick up a factor of two here on all coordintes, but
// because of projectivity, it doesn't matter
func _R2toR1(P r2) (Q r1) {
	Q.X = fp2sub(P.N, P.D)
	Q.Y = fp2add(P.N, P.D)
	Q.Z = P.E
	Q.Ta = fp2mul(dinv, P.F)
	Q.Tb = fp2One
	return
}

func _R2neg(P r2) (Q r2) {
	Q = r2{P.D, P.N, P.E, fp2neg(P.F)}
	return
}

func _R2select(c uint64, P1, P0 r2) (Q r2) {
	Q.N = fp2select(c, P1.N, P0.N)
	Q.D = fp2select(c, P1.D, P0.D)
	Q.E = fp2select(c, P1.E, P0.E)
	Q.F = fp2select(c, P1.F, P0.F)
	return
}

func dbl(P1 r1) (P2 r1) {
	A := fp2sqr(P1.X)
	B := fp2sqr(P1.Y)
	C := fp2mul(fp2Two, fp2sqr(P1.Z))
	D := fp2add(A, B)
	E := fp2sub(fp2sqr(fp2add(P1.X, P1.Y)), D)
	F := fp2sub(B, A)
	G := fp2sub(C, F)
	P2.X = fp2mul(E, G)
	P2.Y = fp2mul(D, F)
	P2.Z = fp2mul(F, G)
	P2.Ta = E
	P2.Tb = D
	return
}

func add_core(P1 r3, P2 r2) (P3 r1) {
	A := fp2mul(P1.D, P2.D)
	B := fp2mul(P1.N, P2.N)
	C := fp2mul(P2.F, P1.T)
	D := fp2mul(P2.E, P1.Z)
	E := fp2sub(B, A)
	F := fp2sub(D, C)
	G := fp2add(D, C)
	H := fp2add(B, A)
	P3.X = fp2mul(E, F)
	P3.Y = fp2mul(G, H)
	P3.Z = fp2mul(F, G)
	P3.Ta = E
	P3.Tb = H
	return
}

func add(P1 r1, P2 r2) (P3 r1) {
	return add_core(_R1toR3(P1), P2)
}

/********** Multiplication without Endomorphisms **********/

func tableWindowed(P r1) (T []r2) {
	Q := dbl(P)
	T = make([]r2, 8)
	T[0] = _R1toR2(P)
	for i := 1; i < 8; i += 1 {
		T[i] = _R1toR2(add(Q, T[i-1]))
	}
	return
}

func mulWindowed(m scalar, P r1, table []r2) (Q r1) {
	// compute [1]P, [3]P, ..., [15]P and negatives
	// XXX: Should check whether this table is for this point
	T := table
	if T == nil {
		T = tableWindowed(P)
	}
	nT := make([]r2, len(T))
	for i, P := range T {
		nT[i] = _R2neg(P)
	}

	// Pre-compute scalars
	d := make([]int, 63)
	reduced := smodN(m)
	odd := reduced[0] & 1
	reduced = sselect(odd, reduced, sadd(reduced, N))
	for i := 0; i < 63; i += 1 {
		d[i] = int(reduced[0]%32) - 16
		reduced = srsh4(ssubi(reduced, d[i]))
	}
	d[62] = int(reduced[0])

	ind := make([]int, len(d))
	sgn := make([]uint64, len(d))
	for i, di := range d {
		a := abs(di)
		ind[i] = (a - 1) / 2
		s := di / a
		sgn[i] = uint64((s + 1) / 2)
	}

	// Compute the product
	Q = _R2toR1(_R2select(sgn[62], T[ind[62]], nT[ind[62]]))
	for i := 61; i >= 0; i -= 1 {
		Q = dbl(dbl(dbl(dbl(Q))))
		Q = add(Q, _R2select(sgn[i], T[ind[i]], nT[ind[i]]))
	}
	return
}

/********** Diffie-Hellman **********/

type mulfn func(scalar, r1, []r2) r1

func dhCore(m scalar, P affine, mul mulfn, table []r2) affine {
	if !pointOnCurve(P.X, P.Y) {
		panic("DH error: Point not on curve")
	}

	P0 := _AffineToR1(P)
	P1 := dbl(dbl(dbl(P0)))
	P2 := dbl(dbl(dbl(dbl(P1))))
	P3 := dbl(P2)
	P3 = add(P3, _R1toR2(P2))
	P3 = add(P3, _R1toR2(P1))
	Q := _R1toAffine(mul(m, P3, table))

	O := affine{Ox, Oy}
	if Q == O {
		panic("DH computation resulted in neutral point")
	}

	return Q
}

func dhWindowed(m scalar, P affine, table []r2) affine {
	return dhCore(m, P, mulWindowed, table)
}
