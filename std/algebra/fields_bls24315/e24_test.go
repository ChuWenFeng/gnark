/*
Copyright © 2020 ConsenSys

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fields_bls24315

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	bls24315 "github.com/consensys/gnark-crypto/ecc/bls24-315"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/test"
)

//--------------------------------------------------------------------
// test

type fp24Add struct {
	A, B E24
	C    E24 `gnark:",public"`
}

func (circuit *fp24Add) Define(api frontend.API) error {
	expected := E24{}
	expected.Add(api, circuit.A, circuit.B)
	expected.MustBeEqual(api, circuit.C)
	return nil
}

func TestAddFp24(t *testing.T) {

	var circuit, witness fp24Add

	// witness values
	var a, b, c bls24315.E24
	a.SetRandom()
	b.SetRandom()
	c.Add(&a, &b)

	witness.A.Assign(&a)
	witness.B.Assign(&b)
	witness.C.Assign(&c)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))
}

type fp24Sub struct {
	A, B E24
	C    E24 `gnark:",public"`
}

func (circuit *fp24Sub) Define(api frontend.API) error {
	expected := E24{}
	expected.Sub(api, circuit.A, circuit.B)
	expected.MustBeEqual(api, circuit.C)
	return nil
}

func TestSubFp24(t *testing.T) {

	var circuit, witness fp24Sub

	// witness values
	var a, b, c bls24315.E24
	a.SetRandom()
	b.SetRandom()
	c.Sub(&a, &b)

	witness.A.Assign(&a)
	witness.B.Assign(&b)
	witness.C.Assign(&c)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))
}

type fp24Mul struct {
	A, B E24
	C    E24 `gnark:",public"`
}

func (circuit *fp24Mul) Define(api frontend.API) error {
	expected := E24{}
	ext := GetBLS24315ExtensionFp24(api)
	expected.Mul(api, circuit.A, circuit.B, ext)
	expected.MustBeEqual(api, circuit.C)
	return nil
}

func TestMulFp24(t *testing.T) {

	var circuit, witness fp24Mul

	// witness values
	var a, b, c bls24315.E24
	a.SetRandom()
	b.SetRandom()
	c.Mul(&a, &b)

	witness.A.Assign(&a)
	witness.B.Assign(&b)
	witness.C.Assign(&c)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))
}

type fp24Square struct {
	A E24
	B E24 `gnark:",public"`
}

func (circuit *fp24Square) Define(api frontend.API) error {
	ext := GetBLS24315ExtensionFp24(api)
	s := circuit.A.Square(api, circuit.A, ext)
	s.MustBeEqual(api, circuit.B)
	return nil
}

func TestSquareFp24(t *testing.T) {

	var circuit, witness fp24Square

	// witness values
	var a, b bls24315.E24
	a.SetRandom()
	b.Square(&a)

	witness.A.Assign(&a)
	witness.B.Assign(&b)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))

}

type fp24CycloSquare struct {
	A E24
	B E24 `gnark:",public"`
}

func (circuit *fp24CycloSquare) Define(api frontend.API) error {
	ext := GetBLS24315ExtensionFp24(api)
	var u, v E24
	u.Square(api, circuit.A, ext)
	v.CyclotomicSquare(api, circuit.A, ext)
	u.MustBeEqual(api, v)
	u.MustBeEqual(api, circuit.B)
	return nil
}

func TestFp24CyclotomicSquare(t *testing.T) {

	var circuit, witness fp24CycloSquare

	// witness values
	var a, b bls24315.E24
	a.SetRandom()

	// put a in the cyclotomic subgroup (we assume the group is Fp24, field of definition of bls24-315)
	var tmp bls24315.E24
	tmp.Conjugate(&a)
	a.Inverse(&a)
	tmp.Mul(&tmp, &a)
	a.FrobeniusQuad(&tmp).Mul(&a, &tmp)

	b.CyclotomicSquare(&a)
	witness.A.Assign(&a)
	witness.B.Assign(&b)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))

}

type fp24CycloSquareCompressed struct {
	A E24
	B E24 `gnark:",public"`
}

func (circuit *fp24CycloSquareCompressed) Define(api frontend.API) error {
	ext := GetBLS24315ExtensionFp24(api)
	var u, v E24
	u.Square(api, circuit.A, ext)
	v.CyclotomicSquareCompressed(api, circuit.A, ext)
	v.Decompress(api, v, ext)
	u.MustBeEqual(api, v)
	u.MustBeEqual(api, circuit.B)
	return nil
}

func TestFp24CyclotomicSquareCompressed(t *testing.T) {

	var circuit, witness fp24CycloSquareCompressed

	// witness values
	var a, b bls24315.E24
	a.SetRandom()

	// put a in the cyclotomic subgroup (we assume the group is Fp24, field of definition of bls24-315)
	var tmp bls24315.E24
	tmp.Conjugate(&a)
	a.Inverse(&a)
	tmp.Mul(&tmp, &a)
	a.FrobeniusQuad(&tmp).Mul(&a, &tmp)

	b.CyclotomicSquare(&a)
	b.Decompress(&b)
	witness.A.Assign(&a)
	witness.B.Assign(&b)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))

}

type fp24Conjugate struct {
	A E24
	C E24 `gnark:",public"`
}

func (circuit *fp24Conjugate) Define(api frontend.API) error {
	expected := E24{}
	expected.Conjugate(api, circuit.A)
	expected.MustBeEqual(api, circuit.C)
	return nil
}

func TestConjugateFp24(t *testing.T) {

	var circuit, witness fp24Conjugate

	// witness values
	var a, c bls24315.E24
	a.SetRandom()
	c.Conjugate(&a)

	witness.A.Assign(&a)
	witness.C.Assign(&c)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))
}

type fp24Inverse struct {
	A E24
	C E24 `gnark:",public"`
}

func (circuit *fp24Inverse) Define(api frontend.API) error {
	expected := E24{}
	ext := GetBLS24315ExtensionFp24(api)
	expected.Inverse(api, circuit.A, ext)
	expected.MustBeEqual(api, circuit.C)
	return nil
}

func TestInverseFp24(t *testing.T) {

	var circuit, witness fp24Inverse

	// witness values
	var a, c bls24315.E24
	a.SetRandom()
	c.Inverse(&a)

	witness.A.Assign(&a)
	witness.C.Assign(&c)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))
}

type fp24MulBy034 struct {
	A    E24 `gnark:",public"`
	W    E24
	B, C E4
}

func (circuit *fp24MulBy034) Define(api frontend.API) error {
	ext := GetBLS24315ExtensionFp24(api)
	circuit.A.MulBy034(api, circuit.B, circuit.C, ext)
	circuit.A.MustBeEqual(api, circuit.W)
	return nil
}

func TestFp24MulBy034(t *testing.T) {

	var circuit, witness fp24MulBy034

	var a bls24315.E24
	var b, c, one bls24315.E4
	one.SetOne()
	a.SetRandom()
	witness.A.Assign(&a)

	b.SetRandom()
	witness.B.Assign(&b)

	c.SetRandom()
	witness.C.Assign(&c)

	a.MulBy034(&one, &b, &c)

	witness.W.Assign(&a)

	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))

}

type fp24Frobenius struct {
	A       E24
	C, D, E E24 `gnark:",public"`
}

func (circuit *fp24Frobenius) Define(api frontend.API) error {
	ext := GetBLS24315ExtensionFp24(api)

	fb := E24{}
	fb.Frobenius(api, circuit.A, ext)
	fb.MustBeEqual(api, circuit.C)

	fbSquare := E24{}
	fbSquare.FrobeniusSquare(api, circuit.A, ext)
	fbSquare.MustBeEqual(api, circuit.D)

	fbQuad := E24{}
	fbQuad.FrobeniusQuad(api, circuit.A, ext)
	fbQuad.MustBeEqual(api, circuit.E)

	return nil
}

func TestFrobeniusFp24(t *testing.T) {

	var circuit, witness fp24Frobenius

	// witness values
	var a, c, d, e bls24315.E24
	a.SetRandom()
	c.Frobenius(&a)
	d.FrobeniusSquare(&a)
	e.FrobeniusQuad(&a)

	witness.A.Assign(&a)
	witness.C.Assign(&c)
	witness.D.Assign(&d)
	witness.E.Assign(&e)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))
}

type fp24FinalExpo struct {
	A E24
	C E24 `gnark:",public"`
}

func (circuit *fp24FinalExpo) Define(api frontend.API) error {
	expected := E24{}
	ext := GetBLS24315ExtensionFp24(api)
	expo := uint64(3218079743)
	expected.FinalExponentiation(api, circuit.A, expo, ext)
	expected.MustBeEqual(api, circuit.C)
	return nil
}

func TestExpFinalExpoFp24(t *testing.T) {
	var circuit, witness fp24FinalExpo

	// witness values
	var a, c bls24315.E24

	a.SetRandom()
	c = bls24315.FinalExponentiation(&a)

	witness.A.Assign(&a)
	witness.C.Assign(&c)

	// cs values
	assert := test.NewAssert(t)
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BW6_633))
}

// benches
var ccsBench frontend.CompiledConstraintSystem

func BenchmarkMulE24(b *testing.B) {
	var c fp24Mul
	b.Run("groth16", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ccsBench, _ = frontend.Compile(ecc.BW6_633, backend.GROTH16, &c)
		}

	})
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkSquareE24(b *testing.B) {
	var c fp24Square
	b.Run("groth16", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ccsBench, _ = frontend.Compile(ecc.BW6_633, backend.GROTH16, &c)
		}

	})
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkInverseE24(b *testing.B) {
	var c fp24Inverse
	b.Run("groth16", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ccsBench, _ = frontend.Compile(ecc.BW6_633, backend.GROTH16, &c)
		}

	})
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkConjugateE24(b *testing.B) {
	var c fp24Conjugate
	b.Run("groth16", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ccsBench, _ = frontend.Compile(ecc.BW6_633, backend.GROTH16, &c)
		}

	})
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkMulBy034E24(b *testing.B) {
	var c fp24MulBy034
	b.Run("groth16", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ccsBench, _ = frontend.Compile(ecc.BW6_633, backend.GROTH16, &c)
		}

	})
	b.Log("groth16", ccsBench.GetNbConstraints())
}
