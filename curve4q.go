package curve4q

import (
	"encoding/binary"
	"fmt"
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

func _R1toR4(P r1) r4 {
	return r4{P.X, P.Y, P.Z}
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

/********** Endomorphisms **********/

var (
	ctau     = fp2elt{fpelt{0x74dcd57cebce74c3, 0x1964de2c3afad20c}, fpelt{0x0000000000000012, 0x000000000000000c}}
	ctaudual = fp2elt{fpelt{0x9ecaa6d9decdf034, 0x4aa740eb23058652}, fpelt{0x0000000000000011, 0x7ffffffffffffff4}}
	cphi0    = fp2elt{fpelt{0xfffffffffffffff7, 0x0000000000000005}, fpelt{0x4f65536cef66f81a, 0x2553a0759182c329}}
	cphi1    = fp2elt{fpelt{0x0000000000000007, 0x0000000000000005}, fpelt{0x334d90e9e28296f9, 0x62c8caa0c50c62cf}}
	cphi2    = fp2elt{fpelt{0x0000000000000015, 0x000000000000000f}, fpelt{0x2c2cb7154f1df391, 0x78df262b6c9b5c98}}
	cphi3    = fp2elt{fpelt{0x0000000000000003, 0x0000000000000002}, fpelt{0x92440457a7962ea4, 0x5084c6491d76342a}}
	cphi4    = fp2elt{fpelt{0x0000000000000003, 0x0000000000000003}, fpelt{0xa1098c923aec6855, 0x12440457a7962ea4}}
	cphi5    = fp2elt{fpelt{0x000000000000000f, 0x000000000000000a}, fpelt{0x669b21d3c5052df3, 0x459195418a18c59e}}
	cphi6    = fp2elt{fpelt{0x0000000000000018, 0x0000000000000012}, fpelt{0xcd3643a78a0a5be7, 0x0b232a8314318b3c}}
	cphi7    = fp2elt{fpelt{0x0000000000000023, 0x0000000000000018}, fpelt{0x66c183035f48781a, 0x3963bc1c99e2ea1a}}
	cphi8    = fp2elt{fpelt{0x00000000000000f0, 0x00000000000000aa}, fpelt{0x44e251582b5d0ef0, 0x1f529f860316cbe5}}
	cphi9    = fp2elt{fpelt{0x0000000000000bef, 0x0000000000000870}, fpelt{0x014d3e48976e2505, 0x0fd52e9cfe00375b}}
	cpsi1    = fp2elt{fpelt{0xedf07f4767e346ef, 0x2af99e9a83d54a02}, fpelt{0x000000000000013a, 0x00000000000000de}}
	cpsi2    = fp2elt{fpelt{0x0000000000000143, 0x00000000000000e4}, fpelt{0x4c7deb770e03f372, 0x21b8d07b99a81f03}}
	cpsi3    = fp2elt{fpelt{0x0000000000000009, 0x0000000000000006}, fpelt{0x3a6e6abe75e73a61, 0x4cb26f161d7d6906}}
	cpsi4    = fp2elt{fpelt{0xfffffffffffffff6, 0x7ffffffffffffff9}, fpelt{0xc59195418a18c59e, 0x334d90e9e28296f9}}
)

func tau(P r4) (Q r4) {
	A := fp2sqr(P.X)
	B := fp2sqr(P.Y)
	C := fp2add(A, B)
	D := fp2sub(A, B)
	Q.X = fp2mul(fp2mul(fp2mul(ctau, P.X), P.Y), D)
	Q.Y = fp2neg(fp2mul(fp2add(fp2mul(fp2Two, fp2sqr(P.Z)), D), C))
	Q.Z = fp2mul(C, D)
	return
}

func tauDual(P r4) (Q r1) {
	A := fp2sqr(P.X)
	B := fp2sqr(P.Y)
	C := fp2add(A, B)
	Q.Ta = fp2sub(B, A)
	D := fp2sub(fp2mul(fp2Two, fp2sqr(P.Z)), Q.Ta)
	Q.Tb = fp2mul(fp2mul(ctaudual, P.X), P.Y)
	Q.X = fp2mul(Q.Tb, C)
	Q.Y = fp2mul(Q.Ta, D)
	Q.Z = fp2mul(C, D)
	return
}

func upsilon(P r4) (Q r4) {
	A := fp2mul(fp2mul(cphi0, P.X), P.Y)
	B := fp2mul(P.Y, P.Z)
	C := fp2sqr(P.Y)
	D := fp2sqr(P.Z)
	F := fp2sqr(D)
	G := fp2sqr(B)
	H := fp2sqr(C)
	I := fp2mul(cphi1, B)
	J := fp2add(C, fp2mul(cphi2, D))
	K := fp2add(fp2add(fp2mul(cphi8, G), H), fp2mul(cphi9, F))
	Q.X = fp2mul(fp2add(I, J), fp2sub(I, J))
	Q.X = fp2conj(fp2mul(fp2mul(A, K), Q.X))
	L := fp2add(C, fp2mul(cphi4, D))
	M := fp2mul(cphi3, B)
	N := fp2mul(fp2add(L, M), fp2sub(L, M))
	Q.Y = fp2add(fp2add(H, fp2mul(cphi6, G)), fp2mul(cphi7, F))
	Q.Y = fp2conj(fp2mul(fp2mul(fp2mul(cphi5, D), N), Q.Y))
	Q.Z = fp2conj(fp2mul(fp2mul(B, K), N))
	return
}

func chi(P r4) (Q r4) {
	A := fp2conj(P.X)
	B := fp2conj(P.Y)
	C := fp2sqr(fp2conj(P.Z))
	D := fp2sqr(A)
	//F := fp2sqr(B) /* Compiler says this is defined and not used */
	G := fp2mul(B, fp2add(D, fp2mul(cpsi2, C)))
	H := fp2neg(fp2add(D, fp2mul(cpsi4, C)))
	Q.X = fp2mul(fp2mul(fp2mul(cpsi1, A), C), H)
	Q.Y = fp2mul(G, fp2add(D, fp2mul(cpsi3, C)))
	Q.Z = fp2mul(G, H)
	return
}

func phi(P r1) r1 {
	return tauDual(upsilon(tau(_R1toR4(P))))
}

func psi(P r1) r1 {
	return tauDual(chi(tau(_R1toR4(P))))
}

/********** Recoding **********/

var (
	b1 = scalar{0x0906ff27e0a0a196, 0x1363e862c22a2da0, 0x07426031ecc8030f, 0x084f739986b9e651}
	b2 = scalar{0x1d495bea84fcc2d4, 0x0000000000000001, 0x0000000000000001, 0x25dbc5bc8dd167d0}
	b3 = scalar{0x17abad1d231f0302, 0x02c4211ae388da51, 0x2e4d21c98927c49f, 0x0a9e6f44c02ecd97}
	b4 = scalar{0x136e340a9108c83f, 0x3122df2dc3e0ff32, 0x068a49f02aa8a9b5, 0x18d5087896de0aea}
	c  = scalar{0x72482c5251a4559c, 0x59f95b0add276f6c, 0x7dd2d17c4625fa78, 0x6bc57def56ce8877}

	L1 = scalar{0x259686e09d1a7d4f, 0xf75682ace6a6bd66, 0xfc5bb5c5ea2be5df, 0x7}
	L2 = scalar{0xd1ba1d84dd627afb, 0x2bd235580f468d8d, 0x8fd4b04caa6c0f8a, 0x3}
	L3 = scalar{0x9b291a33678c203c, 0xc42bd6c965dca902, 0xd038bf8d0bffbaf6, 0x0}
	L4 = scalar{0x12e5666b77e7fdc0, 0x81cbdc3714983d82, 0x1b073877a22d8410, 0x3}
)

func decompose(m scalar) (a scalar) {
	t1 := smultrunc(m, L1)
	t2 := smultrunc(m, L2)
	t3 := smultrunc(m, L3)
	t4 := smultrunc(m, L4)

	temp := m[0] - t1*b1[0] - t2*b2[0] - t3*b3[0] - t4*b4[0] + c[0]
	mask := ^(uint64(0) - (temp & 1))

	a[0] = temp + (mask & b4[0])
	a[1] = t1*b1[1] + t2*b2[1] - t3*b3[1] - t4*b4[1] + c[1] + (mask & b4[1])
	a[2] = t3*b3[2] - t1*b1[2] - t2*b2[2] + t4*b4[2] + c[2] - (mask & b4[2])
	a[3] = t1*b1[3] - t2*b2[3] - t3*b3[3] + t4*b4[3] + c[3] - (mask & b4[3])
	return
}

func recode(v scalar) (m []uint64, d []uint64) {
	bit := func(x uint64, n uint) uint64 {
		return (x >> n) & 1
	}

	d = make([]uint64, 65)
	m = make([]uint64, 65)
	for i := uint(0); i < 64; i += 1 {
		b1 := bit(v[0], i+1)
		d[i] = 0
		m[i] = b1

		for _, j := range []uint{1, 2, 3} {
			bj := bit(v[j], 0)
			d[i] += bj << uint(j-1)
			c := (b1 | bj) ^ b1
			v[j] = (v[j] >> 1) + c
		}
	}

	fmt.Println()

	d[64] = v[1] + 2*v[2] + 4*v[3]
	m[64] = 1
	return
}

/********** Optimized multiplication **********/

func tableEndo(P r1) (T []r2) {
	Q1 := phi(P)
	R1 := psi(P)
	S1 := psi(Q1)

	Q := _R1toR3(Q1)
	R := _R1toR3(R1)
	S := _R1toR3(S1)

	T = make([]r2, 8)
	T[0] = _R1toR2(P)                 // P
	T[1] = _R1toR2(add_core(Q, T[0])) // P + Q
	T[2] = _R1toR2(add_core(R, T[0])) // P + R
	T[3] = _R1toR2(add_core(R, T[1])) // P + Q + R
	T[4] = _R1toR2(add_core(S, T[0])) // P + S
	T[5] = _R1toR2(add_core(S, T[1])) // P + Q + S
	T[6] = _R1toR2(add_core(S, T[2])) // P + R + S
	T[7] = _R1toR2(add_core(S, T[3])) // P + Q + R + S
	return
}

func mulEndo(m scalar, P r1, table []r2) (Q r1) {
	T := table
	if T == nil {
		T = tableEndo(P)
	}
	nT := make([]r2, len(T))
	for i, P := range T {
		nT[i] = _R2neg(P)
	}

	// Pre-compute scalars
	scalars := decompose(m)
	s, d := recode(scalars)

	// Compute the product
	Q = _R2toR1(_R2select(s[64], T[d[64]], nT[d[64]]))
	for i := 61; i >= 0; i -= 1 {
		Q = dbl(Q)
		Q = add(Q, _R2select(s[i], T[d[i]], nT[d[i]]))
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
