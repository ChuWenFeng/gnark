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

package bandersnatch

import (
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/twistededwards/bandersnatch"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/test"
)

type mustBeOnCurve struct {
	P Point
}

func (circuit *mustBeOnCurve) Define(api frontend.API) error {

	// get edwards curve params
	params, err := NewEdCurve(api.Curve())
	if err != nil {
		return err
	}

	circuit.P.MustBeOnCurve(api, params)

	return nil
}

func TestIsOnCurve(t *testing.T) {

	assert := test.NewAssert(t)

	var circuit, witness mustBeOnCurve

	params, err := NewEdCurve(ecc.BLS12_381)
	if err != nil {
		t.Fatal(err)
	}

	witness.P.X = (params.Base.X)
	witness.P.Y = (params.Base.Y)

	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BLS12_381))

}

type add struct {
	P, E Point
}

func (circuit *add) Define(api frontend.API) error {

	// get edwards curve params
	params, err := NewEdCurve(api.Curve())
	if err != nil {
		return err
	}

	p := Point{}
	p.X = params.Base.X
	p.Y = params.Base.Y
	res := circuit.P.Add(api, &circuit.P, &p, params)

	api.AssertIsEqual(res.X, circuit.E.X)
	api.AssertIsEqual(res.Y, circuit.E.Y)

	return nil
}

func TestAddFixedPoint(t *testing.T) {

	assert := test.NewAssert(t)

	var circuit, witness add

	// generate a random point, and compute expected_point = base + random_point
	params, err := NewEdCurve(ecc.BLS12_381)
	if err != nil {
		t.Fatal(err)
	}
	var base, point, expected bandersnatch.PointAffine
	base.X.SetBigInt(&params.Base.X)
	base.Y.SetBigInt(&params.Base.Y)
	point.Set(&base)
	r := big.NewInt(5)
	point.ScalarMul(&point, r)
	expected.Add(&base, &point)

	// populate witness
	witness.P.X = (point.X.String())
	witness.P.Y = (point.Y.String())
	witness.E.X = (expected.X.String())
	witness.E.Y = (expected.Y.String())

	// creates r1cs
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BLS12_381))

}

//-------------------------------------------------------------
// addGeneric

type addGeneric struct {
	P1, P2, E Point
}

func (circuit *addGeneric) Define(api frontend.API) error {

	// get edwards curve params
	params, err := NewEdCurve(api.Curve())
	if err != nil {
		return err
	}

	res := circuit.P1.Add(api, &circuit.P1, &circuit.P2, params)

	api.AssertIsEqual(res.X, circuit.E.X)
	api.AssertIsEqual(res.Y, circuit.E.Y)

	return nil
}

func TestAddGeneric(t *testing.T) {

	assert := test.NewAssert(t)
	var circuit, witness addGeneric

	// generate random points, and compute expected_point = point1 + point2s
	params, err := NewEdCurve(ecc.BLS12_381)
	if err != nil {
		t.Fatal(err)
	}
	var point1, point2, expected bandersnatch.PointAffine
	point1.X.SetBigInt(&params.Base.X)
	point1.Y.SetBigInt(&params.Base.Y)
	point2.Set(&point1)
	r1 := big.NewInt(5)
	r2 := big.NewInt(12)
	point1.ScalarMul(&point1, r1)
	point2.ScalarMul(&point2, r2)
	expected.Add(&point1, &point2)

	// populate witness
	witness.P1.X = (point1.X.String())
	witness.P1.Y = (point1.Y.String())
	witness.P2.X = (point2.X.String())
	witness.P2.Y = (point2.Y.String())
	witness.E.X = (expected.X.String())
	witness.E.Y = (expected.Y.String())

	// creates r1cs
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BLS12_381))

}

//-------------------------------------------------------------
// Double
type double struct {
	P, E Point
}

func (circuit *double) Define(api frontend.API) error {

	// get edwards curve params
	params, err := NewEdCurve(api.Curve())
	if err != nil {
		return err
	}

	res := circuit.P.Double(api, &circuit.P, params)

	api.AssertIsEqual(res.X, circuit.E.X)
	api.AssertIsEqual(res.Y, circuit.E.Y)

	return nil
}

func TestDouble(t *testing.T) {

	assert := test.NewAssert(t)

	var circuit, witness double

	// generate witness data
	params, err := NewEdCurve(ecc.BLS12_381)
	if err != nil {
		t.Fatal(err)
	}
	var base, expected bandersnatch.PointAffine
	base.X.SetBigInt(&params.Base.X)
	base.Y.SetBigInt(&params.Base.Y)
	expected.Double(&base)

	// populate witness
	witness.P.X = (base.X.String())
	witness.P.Y = (base.Y.String())
	witness.E.X = (expected.X.String())
	witness.E.Y = (expected.Y.String())

	// creates r1cs
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BLS12_381))

}

type scalarMulFixed struct {
	E Point
	S frontend.Variable
}

func (circuit *scalarMulFixed) Define(api frontend.API) error {

	// get edwards curve params
	params, err := NewEdCurve(api.Curve())
	if err != nil {
		return err
	}

	var resFixed, p Point
	p.X = params.Base.X
	p.Y = params.Base.Y
	resFixed.ScalarMul(api, &p, circuit.S, params)

	api.AssertIsEqual(resFixed.X, circuit.E.X)
	api.AssertIsEqual(resFixed.Y, circuit.E.Y)

	return nil
}

func TestScalarMulFixed(t *testing.T) {

	assert := test.NewAssert(t)

	var circuit, witness scalarMulFixed

	// generate witness data
	params, err := NewEdCurve(ecc.BLS12_381)
	if err != nil {
		t.Fatal(err)
	}
	var base, expected bandersnatch.PointAffine
	base.X.SetBigInt(&params.Base.X)
	base.Y.SetBigInt(&params.Base.Y)
	r := big.NewInt(928323002)
	expected.ScalarMul(&base, r)

	// populate witness
	witness.E.X = (expected.X.String())
	witness.E.Y = (expected.Y.String())
	witness.S = (r)

	// creates r1cs
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BLS12_381))

}

type scalarMulGeneric struct {
	P, E Point
	S    frontend.Variable
}

func (circuit *scalarMulGeneric) Define(api frontend.API) error {

	// get edwards curve params
	params, err := NewEdCurve(api.Curve())
	if err != nil {
		return err
	}

	resGeneric := circuit.P.ScalarMul(api, &circuit.P, circuit.S, params)

	api.AssertIsEqual(resGeneric.X, circuit.E.X)
	api.AssertIsEqual(resGeneric.Y, circuit.E.Y)

	return nil
}

func TestScalarMulGeneric(t *testing.T) {

	assert := test.NewAssert(t)

	var circuit, witness scalarMulGeneric

	// generate witness data
	params, err := NewEdCurve(ecc.BLS12_381)
	if err != nil {
		t.Fatal(err)
	}
	var base, point, expected bandersnatch.PointAffine
	base.X.SetBigInt(&params.Base.X)
	base.Y.SetBigInt(&params.Base.Y)
	s := big.NewInt(902)
	point.ScalarMul(&base, s) // random point
	r := big.NewInt(230928302)
	expected.ScalarMul(&point, r)

	// populate witness
	witness.P.X = (point.X.String())
	witness.P.Y = (point.Y.String())
	witness.E.X = (expected.X.String())
	witness.E.Y = (expected.Y.String())
	witness.S = (r)

	// creates r1cs
	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BLS12_381))

}

type neg struct {
	P, E Point
}

func (circuit *neg) Define(api frontend.API) error {

	circuit.P.Neg(api, &circuit.P)
	api.AssertIsEqual(circuit.P.X, circuit.E.X)
	api.AssertIsEqual(circuit.P.Y, circuit.E.Y)

	return nil
}

func TestNeg(t *testing.T) {

	assert := test.NewAssert(t)

	// generate witness data
	params, err := NewEdCurve(ecc.BLS12_381)
	if err != nil {
		t.Fatal(err)
	}
	var base, expected bandersnatch.PointAffine
	base.X.SetBigInt(&params.Base.X)
	base.Y.SetBigInt(&params.Base.Y)
	expected.Neg(&base)

	// generate witness
	var circuit, witness neg
	witness.P.X = (base.X)
	witness.P.Y = (base.Y)
	witness.E.X = (expected.X)
	witness.E.Y = (expected.Y)

	assert.SolvingSucceeded(&circuit, &witness, test.WithCurves(ecc.BLS12_381))

}

// Bench
func BenchmarkDouble(b *testing.B) {
	var c double
	ccsBench, _ := frontend.Compile(ecc.BLS12_381, backend.GROTH16, &c)
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkAddGeneric(b *testing.B) {
	var c addGeneric
	ccsBench, _ := frontend.Compile(ecc.BLS12_381, backend.GROTH16, &c)
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkAddFixedPoint(b *testing.B) {
	var c add
	ccsBench, _ := frontend.Compile(ecc.BLS12_381, backend.GROTH16, &c)
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkMustBeOnCurve(b *testing.B) {
	var c mustBeOnCurve
	ccsBench, _ := frontend.Compile(ecc.BLS12_381, backend.GROTH16, &c)
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkScalarMulGeneric(b *testing.B) {
	var c scalarMulGeneric
	ccsBench, _ := frontend.Compile(ecc.BLS12_381, backend.GROTH16, &c)
	b.Log("groth16", ccsBench.GetNbConstraints())
}

func BenchmarkScalarMulFixed(b *testing.B) {
	var c scalarMulFixed
	ccsBench, _ := frontend.Compile(ecc.BLS12_381, backend.GROTH16, &c)
	b.Log("groth16", ccsBench.GetNbConstraints())
}
