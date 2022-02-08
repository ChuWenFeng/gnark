// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by gnark DO NOT EDIT

package cs

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"

	"github.com/fxamacker/cbor/v2"

	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/frontend/schema"
	"github.com/consensys/gnark/internal/backend/compiled"
	"github.com/consensys/gnark/internal/backend/ioutils"

	"github.com/consensys/gnark-crypto/ecc"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"

	bn254witness "github.com/consensys/gnark/internal/backend/bn254/witness"
)

// R1CS decsribes a set of R1CS constraint
type R1CS struct {
	compiled.R1CS
	Coefficients []fr.Element // R1C coefficients indexes point here
	layers Layer
}

type Layer struct{
	mulLayerIndex [][]int//第0层表示public+secret,约束从第1层开始
	high int
}

func (l *Layer)GetConstraints()int{
	var sum int
	for _,i := range l.mulLayerIndex{
		sum += len(i)
	}
	return sum
}

func (cs *R1CS)Layers()error{
	// w := witness{}
	// if err := w.FromFullAssignment(witness); err != nil {
	// 	return err
	// }
	inputs := cs.NbPublicVariables + cs.NbSecretVariables
	nbWires := inputs + cs.NbInternalVariables
	solved := make([]bool,nbWires)
	variableLayer := make([]int,nbWires)
	var layer = Layer{
		mulLayerIndex: make([][]int, 2),
		high: 0,
	}
	for i:=0;i<inputs;i++{
		solved[i] = true
		variableLayer[i] = 0
	}
	var loc uint8
	var high = 0
	var termToCompute compiled.Term
	var hintFlag bool = false
	processTerm := func(t compiled.Term, locValue uint8) error {
		vID := t.WireID()

		// wire is already computed, we just accumulate in val
		if solved[vID] {
			h := variableLayer[vID]
			if h > high{
				high = h
			}
			return nil
		}

		// first we check if this is a hint wire
		if hint, ok := cs.MHints[vID]; ok {
			if !hintFlag{
				hintFlag = true
				termToCompute = t
			}
		
			var line compiled.LinearExpression
			input := hint.Inputs[0]//如果有hint.Inputs[1]，则为constant，无需判断
			switch t := input.(type){
			case compiled.Variable: line = t.LinExp
			case compiled.LinearExpression: line = t
			}
			for _,linearexp := range line{ 
			_, viID, visibility := linearexp.Unpack()
			if visibility == schema.Virtual{
				continue
			}
			if solved[viID]{
				h := variableLayer[viID]
				if h > high{
					high = h
				}
			}else{
				return  errors.New("expected wire to be instantiated while evaluating hint")
			}
			
			}
			return nil
		}

		if loc != 0 {
			panic("found more than one wire to instantiate")
		}
		termToCompute = t
		loc = locValue
		return nil
	}
	processHigh := func (i int)  {
		high++
		if high >= len(layer.mulLayerIndex) {
			layer.mulLayerIndex = append(layer.mulLayerIndex, make([]int, 0))
		}
		layer.mulLayerIndex[high] = append(layer.mulLayerIndex[high], i)

		if loc != 0 || hintFlag{
			vID := termToCompute.WireID()
			solved[vID] = true
			variableLayer[vID] = high
		}
		loc = 0
		high = 0
		hintFlag = false
	}
	
	for i:=0;i<len(cs.Constraints);i++{
		r := cs.Constraints[i]
		for _,t := range r.L.LinExp{
			if err := processTerm(t,1);err != nil{
				return err
			}
		}

		for _, t := range r.R.LinExp {
			if err := processTerm(t, 2); err != nil {
				return err
			}
		}
	
		for _, t := range r.O.LinExp {
			if err := processTerm(t, 3); err != nil {
				return err
			}
		}
		processHigh(i)

	}

	layer.high = len(layer.mulLayerIndex)-1
	
	lc := layer.GetConstraints()
	if lc != len(cs.Constraints){
		return errors.New("layer constraints not equal cs constraints")
	}
	for i,f := range solved{
		if !f{
			return	errors.New(fmt.Sprint("VID:",i," solved not finish"))
		}
	}

	cs.layers = layer

	return nil
}

// NewR1CS returns a new R1CS and sets cs.Coefficient (fr.Element) from provided big.Int values
func NewR1CS(cs compiled.R1CS, coefficients []big.Int) *R1CS {
	r := R1CS{
		R1CS:         cs,
		Coefficients: make([]fr.Element, len(coefficients)),
	}
	for i := 0; i < len(coefficients); i++ {
		r.Coefficients[i].SetBigInt(&coefficients[i])
	}
	err := r.Layers()
	if err != nil{
		panic(err)
	}
	return &r
}

// Solve sets all the wires and returns the a, b, c vectors.
// the cs system should have been compiled before. The entries in a, b, c are in Montgomery form.
// a, b, c vectors: ab-c = hz
// witness = [publicWires | secretWires] (without the ONE_WIRE !)
// returns  [publicWires | secretWires | internalWires ]
func (cs *R1CS) Solve(witness, a, b, c []fr.Element, opt backend.ProverConfig) ([]fr.Element, error) {

	nbWires := cs.NbPublicVariables + cs.NbSecretVariables + cs.NbInternalVariables
	solution, err := newSolution(nbWires, opt.HintFunctions, cs.Coefficients)
	if err != nil {
		return make([]fr.Element, nbWires), err
	}

	if len(witness) != int(cs.NbPublicVariables-1+cs.NbSecretVariables) { // - 1 for ONE_WIRE
		return solution.values, fmt.Errorf("invalid witness size, got %d, expected %d = %d (public - ONE_WIRE) + %d (secret)", len(witness), int(cs.NbPublicVariables-1+cs.NbSecretVariables), cs.NbPublicVariables-1, cs.NbSecretVariables)
	}

	// compute the wires and the a, b, c polynomials
	if len(a) != len(cs.Constraints) || len(b) != len(cs.Constraints) || len(c) != len(cs.Constraints) {
		return solution.values, errors.New("invalid input size: len(a, b, c) == len(Constraints)")
	}

	solution.solved[0] = true // ONE_WIRE
	solution.values[0].SetOne()
	copy(solution.values[1:], witness) // TODO factorize
	for i := 0; i < len(witness); i++ {
		solution.solved[i+1] = true
	}

	// keep track of the number of wire instantiations we do, for a sanity check to ensure
	// we instantiated all wires
	solution.nbSolved += len(witness) + 1

	// now that we know all inputs are set, defer log printing once all solution.values are computed
	// (or sooner, if a constraint is not satisfied)
	defer solution.printLogs(opt.LoggerOut, cs.Logs)

	// check if there is an inconsistant constraint
	// var check fr.Element

	// for each constraint
	// we are guaranteed that each R1C contains at most one unsolved wire
	// first we solve the unsolved wire (if any)
	// then we check that the constraint is valid
	// if a[i] * b[i] != c[i]; it means the constraint is not satisfied
	// for i := 0; i < len(cs.Constraints); i++ {
	// 	// solve the constraint, this will compute the missing wire of the gate
	// 	if err := cs.solveConstraint(cs.Constraints[i], &solution); err != nil {
	// 		if dID, ok := cs.MDebug[i]; ok {
	// 			debugInfoStr := solution.logValue(cs.DebugInfo[dID])
	// 			return solution.values, fmt.Errorf("%w: %s", err, debugInfoStr)
	// 		}
	// 		return solution.values, err
	// 	}

	// 	// compute values for the R1C (ie value * coeff)
	// 	a[i], b[i], c[i] = cs.instantiateR1C(cs.Constraints[i], &solution)

	// 	// ensure a[i] * b[i] == c[i]
	// 	check.Mul(&a[i], &b[i])
	// 	if !check.Equal(&c[i]) {
	// 		errMsg := fmt.Sprintf("%s ⋅ %s != %s", a[i].String(), b[i].String(), c[i].String())
	// 		if dID, ok := cs.MDebug[i]; ok {
	// 			errMsg = solution.logValue(cs.DebugInfo[dID])
	// 		}
	// 		return solution.values, fmt.Errorf("constraint #%d is not satisfied: %s", i, errMsg)
	// 	}
	// }

	// // sanity check; ensure all wires are marked as "instantiated"
	// if !solution.isValid() {
	// 	panic("solver didn't instantiate all wires")
	// }

	// return solution.values, nil

	type csErr struct{
		idx int
		err error
		t int//0 means solve err, 1 means check err
	}
	var wg sync.WaitGroup
	parallsolveConstraint := func(idx int,solveErrchan chan csErr){
		defer wg.Done()
		var check fr.Element
		if err := cs.solveConstraint(cs.Constraints[idx],&solution);err != nil{
			solveErrchan <- csErr{idx:idx,err:err,t:0}
		}
		a[idx],b[idx],c[idx] = cs.instantiateR1C(cs.Constraints[idx],&solution)
		check.Mul(&a[idx],&b[idx])
		if !check.Equal(&c[idx]){
			solveErrchan <- csErr{idx:idx,err:err,t:1}
		}
	}

	for i,layer := range cs.layers.mulLayerIndex{
		solveErrchan := make(chan csErr,20)
		for _,csIdx := range layer{
			wg.Add(1)
			go parallsolveConstraint(csIdx,solveErrchan)
		}
		go func(){
			wg.Wait()
			close(solveErrchan)
		}()
		
		for ce := range solveErrchan{
			fmt.Println(i)
			if ce.t == 0{
				if dID, ok := cs.MDebug[ce.idx]; ok {
					debugInfoStr := solution.logValue(cs.DebugInfo[dID])
					return solution.values, fmt.Errorf("%w: %s", err, debugInfoStr)
				}
				return solution.values, err
			}else{
				errMsg := fmt.Sprintf("%s ⋅ %s != %s", a[ce.idx].String(), b[ce.idx].String(), c[ce.idx].String())
				if dID, ok := cs.MDebug[ce.idx]; ok {
					errMsg = solution.logValue(cs.DebugInfo[dID])
				}
				return solution.values, fmt.Errorf("constraint #%d is not satisfied: %s", i, errMsg)
			}
		}
		

	}

	// this check need put a lock in solution.set function, this will super slow, so i give up ckeck it
	// sanity check; ensure all wires are marked as "instantiated"
	// if !solution.isValid() {
	// 	panic("solver didn't instantiate all wires")
	// }

	return solution.values, nil
}

// IsSolved returns nil if given witness solves the R1CS and error otherwise
// this method wraps cs.Solve() and allocates cs.Solve() inputs
func (cs *R1CS) IsSolved(witness *witness.Witness, opts ...backend.ProverOption) error {
	opt, err := backend.NewProverConfig(opts...)
	if err != nil {
		return err
	}

	a := make([]fr.Element, len(cs.Constraints))
	b := make([]fr.Element, len(cs.Constraints))
	c := make([]fr.Element, len(cs.Constraints))
	v := witness.Vector.(*bn254witness.Witness)
	_, err = cs.Solve(*v, a, b, c, opt)
	return err
}

// mulByCoeff sets res = res * t.Coeff
func (cs *R1CS) mulByCoeff(res *fr.Element, t compiled.Term) {
	cID := t.CoeffID()
	switch cID {
	case compiled.CoeffIdOne:
		return
	case compiled.CoeffIdMinusOne:
		res.Neg(res)
	case compiled.CoeffIdZero:
		res.SetZero()
	case compiled.CoeffIdTwo:
		res.Double(res)
	default:
		res.Mul(res, &cs.Coefficients[cID])
	}
}

// compute left, right, o part of a cs constraint
// this function is called when all the wires have been computed
// it instantiates the l, r o part of a R1C
func (cs *R1CS) instantiateR1C(r compiled.R1C, solution *solution) (a, b, c fr.Element) {
	var v fr.Element
	for _, t := range r.L.LinExp {
		v = solution.computeTerm(t)
		a.Add(&a, &v)
	}
	for _, t := range r.R.LinExp {
		v = solution.computeTerm(t)
		b.Add(&b, &v)
	}
	for _, t := range r.O.LinExp {
		v = solution.computeTerm(t)
		c.Add(&c, &v)
	}
	return
}

// solveR1c computes a wire by solving a cs
// the function searches for the unset wire (either the unset wire is
// alone, or it can be computed without ambiguity using the other computed wires
// , eg when doing a binary decomposition: either way the missing wire can
// be computed without ambiguity because the cs is correctly ordered)
//
// It returns the 1 if the the position to solve is in the quadratic part (it
// means that there is a division and serves to navigate in the log info for the
// computational constraints), and 0 otherwise.
func (cs *R1CS) solveConstraint(r compiled.R1C, solution *solution) error {

	// the index of the non zero entry shows if L, R or O has an uninstantiated wire
	// the content is the ID of the wire non instantiated
	var loc uint8

	var a, b, c fr.Element
	var termToCompute compiled.Term

	processTerm := func(t compiled.Term, val *fr.Element, locValue uint8) error {
		vID := t.WireID()

		// wire is already computed, we just accumulate in val
		if solution.solved[vID] {
			v := solution.computeTerm(t)
			val.Add(val, &v)
			return nil
		}

		// first we check if this is a hint wire
		if hint, ok := cs.MHints[vID]; ok {
			if err := solution.solveWithHint(vID, hint); err != nil {
				return err
			}
			v := solution.computeTerm(t)
			val.Add(val, &v)
			return nil
		}

		if loc != 0 {
			panic("found more than one wire to instantiate")
		}
		termToCompute = t
		loc = locValue
		return nil
	}

	for _, t := range r.L.LinExp {
		if err := processTerm(t, &a, 1); err != nil {
			return err
		}
	}

	for _, t := range r.R.LinExp {
		if err := processTerm(t, &b, 2); err != nil {
			return err
		}
	}

	for _, t := range r.O.LinExp {
		if err := processTerm(t, &c, 3); err != nil {
			return err
		}
	}

	if loc == 0 {
		// there is nothing to solve, may happen if we have an assertion
		// (ie a constraints that doesn't yield any output)
		// or if we solved the unsolved wires with hint functions
		return nil
	}

	// we compute the wire value and instantiate it
	vID := termToCompute.WireID()

	// solver result
	var wire fr.Element

	switch loc {
	case 1:
		if !b.IsZero() {
			wire.Div(&c, &b).
				Sub(&wire, &a)
			cs.mulByCoeff(&wire, termToCompute)
		}
	case 2:
		if !a.IsZero() {
			wire.Div(&c, &a).
				Sub(&wire, &b)
			cs.mulByCoeff(&wire, termToCompute)
		}
	case 3:
		wire.Mul(&a, &b).
			Sub(&wire, &c)
		cs.mulByCoeff(&wire, termToCompute)
	}

	solution.set(vID, wire)

	return nil
}

// GetConstraints return a list of constraint formatted as L⋅R == O
// such that [0] -> L, [1] -> R, [2] -> O
func (cs *R1CS) GetConstraints() [][]string {
	r := make([][]string, 0, len(cs.Constraints))
	for _, c := range cs.Constraints {
		// for each constraint, we build a string representation of it's L, R and O part
		// if we are worried about perf for large cs, we could do a string builder + csv format.
		var line [3]string
		line[0] = cs.vtoString(c.L)
		line[1] = cs.vtoString(c.R)
		line[2] = cs.vtoString(c.O)
		r = append(r, line[:])
	}
	return r
}

func (cs *R1CS) vtoString(l compiled.Variable) string {
	var sbb strings.Builder
	for i := 0; i < len(l.LinExp); i++ {
		cs.termToString(l.LinExp[i], &sbb)
		if i+1 < len(l.LinExp) {
			sbb.WriteString(" + ")
		}
	}
	return sbb.String()
}

func (cs *R1CS) termToString(t compiled.Term, sbb *strings.Builder) {
	tID := t.CoeffID()
	if tID == compiled.CoeffIdOne {
		// do nothing, just print the variable
	} else if tID == compiled.CoeffIdMinusOne {
		// print neg sign
		sbb.WriteByte('-')
	} else if tID == compiled.CoeffIdZero {
		sbb.WriteByte('0')
		return
	} else {
		sbb.WriteString(cs.Coefficients[tID].String())
		sbb.WriteString("⋅")
	}
	vID := t.WireID()
	visibility := t.VariableVisibility()

	switch visibility {
	case schema.Internal:
		if _, isHint := cs.MHints[vID]; isHint {
			sbb.WriteString(fmt.Sprintf("hv%d", vID-cs.NbPublicVariables-cs.NbSecretVariables))
		} else {
			sbb.WriteString(fmt.Sprintf("v%d", vID-cs.NbPublicVariables-cs.NbSecretVariables))
		}
	case schema.Public:
		if vID == 0 {
			sbb.WriteByte('1') // one wire
		} else {
			sbb.WriteString(fmt.Sprintf("p%d", vID-1))
		}
	case schema.Secret:
		sbb.WriteString(fmt.Sprintf("s%d", vID-cs.NbPublicVariables))
	default:
		sbb.WriteString("<?>")
	}
}

// GetNbCoefficients return the number of unique coefficients needed in the R1CS
func (cs *R1CS) GetNbCoefficients() int {
	return len(cs.Coefficients)
}

// CurveID returns curve ID as defined in gnark-crypto
func (cs *R1CS) CurveID() ecc.ID {
	return ecc.BN254
}

// FrSize return fr.Limbs * 8, size in byte of a fr element
func (cs *R1CS) FrSize() int {
	return fr.Limbs * 8
}

// WriteTo encodes R1CS into provided io.Writer using cbor
func (cs *R1CS) WriteTo(w io.Writer) (int64, error) {
	_w := ioutils.WriterCounter{W: w} // wraps writer to count the bytes written
	enc, err := cbor.CoreDetEncOptions().EncMode()
	if err != nil {
		return 0, err
	}
	encoder := enc.NewEncoder(&_w)

	// encode our object
	err = encoder.Encode(cs)
	return _w.N, err
}

// ReadFrom attempts to decode R1CS from io.Reader using cbor
func (cs *R1CS) ReadFrom(r io.Reader) (int64, error) {
	dm, err := cbor.DecOptions{
		MaxArrayElements: 134217728,
		MaxMapPairs:      134217728,
	}.DecMode()

	if err != nil {
		return 0, err
	}
	decoder := dm.NewDecoder(r)
	if err := decoder.Decode(&cs); err != nil {
		return int64(decoder.NumBytesRead()), err
	}

	return int64(decoder.NumBytesRead()), nil
}
