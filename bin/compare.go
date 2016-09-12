package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/bifurcation/curve4q"
	"golang.org/x/crypto/curve25519"
)

const SAMPLES = 1000

func main() {
	var dst [32]byte
	var tic, toc time.Time

	corpus := make([][32]byte, SAMPLES)
	for i := range corpus {
		rand.Read(corpus[i][:])
	}

	tic = time.Now()
	for i := 0; i < SAMPLES; i += 1 {
		curve4q.ScalarBaseMultWin(&dst, &corpus[i])
	}
	toc = time.Now()
	fmt.Printf("curve4q(win):    %v\n", toc.Sub(tic))

	tic = time.Now()
	for i := 0; i < SAMPLES; i += 1 {
		curve4q.ScalarBaseMultEndo(&dst, &corpus[i])
	}
	toc = time.Now()
	fmt.Printf("curve4q(endo):    %v\n", toc.Sub(tic))

	tic = time.Now()
	for i := 0; i < SAMPLES; i += 1 {
		curve25519.ScalarMult(&dst, &corpus[i], &corpus[i])
	}
	toc = time.Now()
	fmt.Printf("curve25519: %v\n", toc.Sub(tic))
}
