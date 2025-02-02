package frontend

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/debug"
	"github.com/consensys/gnark/frontend/schema"
)

var tVariable reflect.Type

func init() {
	tVariable = reflect.ValueOf(struct{ A Variable }{}).FieldByName("A").Type()
}

// Builder represents a constraint system builder
type Builder interface {
	API
	CheckVariables() error
	NewPublicVariable(name string) Variable
	NewSecretVariable(name string) Variable
	Compile() (CompiledConstraintSystem, error)
	SetSchema(*schema.Schema)
}

type NewBuilder func(ecc.ID) (Builder, error)

// Compile will generate a ConstraintSystem from the given circuit
//
// 1. it will first allocate the user inputs (see type Tag for more info)
// example:
// 		type MyCircuit struct {
// 			Y frontend.Variable `gnark:"exponent,public"`
// 		}
// in that case, Compile() will allocate one public variable with id "exponent"
//
// 2. it then calls circuit.Define(curveID, R1CS) to build the internal constraint system
// from the declarative code
//
// 3. finally, it converts that to a ConstraintSystem.
// 		if zkpID == backend.GROTH16	→ R1CS
//		if zkpID == backend.PLONK 	→ SparseR1CS
//
// initialCapacity is an optional parameter that reserves memory in slices
// it should be set to the estimated number of constraints in the circuit, if known.
func Compile(curveID ecc.ID, zkpID backend.ID, circuit Circuit, opts ...CompileOption) (CompiledConstraintSystem, error) {
	// setup option
	opt := compileConfig{}
	for _, o := range opts {
		if err := o(&opt); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}
	newBuilder := opt.newBuilder
	if newBuilder == nil {
		var ok bool
		backendsM.RLock()
		newBuilder, ok = backends[zkpID]
		backendsM.RUnlock()
		if !ok {
			return nil, fmt.Errorf("no default constraint builder registered nor set as option")
		}
	}
	builder, err := newBuilder(curveID)
	if err != nil {
		return nil, fmt.Errorf("new builder: %w", err)
	}

	if err = bootstrap(builder, circuit); err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)

	}

	// ensure all inputs and hints are constrained
	if !opt.ignoreUnconstrainedInputs {
		if err := builder.CheckVariables(); err != nil {
			return nil, err
		}
	}

	ccs, err := builder.Compile()
	if err != nil {
		return nil, fmt.Errorf("compile system: %w", err)
	}
	return ccs, nil
}

func bootstrap(builder Builder, circuit Circuit) (err error) {
	// ensure circuit.Define has pointer receiver
	if reflect.ValueOf(circuit).Kind() != reflect.Ptr {
		return errors.New("frontend.Circuit methods must be defined on pointer receiver")
	}

	// leaf handlers are called when encoutering leafs in the circuit data struct
	// leafs are Constraints that need to be initialized in the context of compiling a circuit
	var handler schema.LeafHandler = func(visibility schema.Visibility, name string, tInput reflect.Value) error {
		if tInput.CanSet() {
			switch visibility {
			case schema.Secret:
				tInput.Set(reflect.ValueOf(builder.NewSecretVariable(name)))
			case schema.Public:
				tInput.Set(reflect.ValueOf(builder.NewPublicVariable(name)))
			case schema.Unset:
				return errors.New("can't set val " + name + " visibility is unset")
			}

			return nil
		}
		return errors.New("can't set val " + name)
	}
	// recursively parse through reflection the circuits members to find all Constraints that need to be allocated
	// (secret or public inputs)
	s, err := schema.Parse(circuit, tVariable, handler)
	if err != nil {
		return err
	}
	builder.SetSchema(s)

	// recover from panics to print user-friendlier messages
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v\n%s", r, debug.Stack())
		}
	}()

	// call Define() to fill in the Constraints
	if err = circuit.Define(builder); err != nil {
		return fmt.Errorf("define circuit: %w", err)
	}

	return
}

// CompileOption defines option for altering the behaviour of the Compile
// method. See the descriptions of the functions returning instances of this
// type for available options.
type CompileOption func(opt *compileConfig) error

type compileConfig struct {
	capacity                  int
	ignoreUnconstrainedInputs bool
	newBuilder                NewBuilder
}

// WithCapacity is a compile option that specifies the estimated capacity needed
// for internal variables and constraints. If not set, then the initial capacity
// is 0 and is dynamically allocated as needed.
func WithCapacity(capacity int) CompileOption {
	return func(opt *compileConfig) error {
		opt.capacity = capacity
		return nil
	}
}

// IgnoreUnconstrainedInputs is a compile option which allow compiling input
// circuits where not all inputs are not constrained. If not set, then the
// compiler returns an error if there exists an unconstrained input.
//
// This option is useful for debugging circuits, but should not be used in
// production settings as it means that there is a potential error in the
// circuit definition or that it is possible to optimize witness size.
func IgnoreUnconstrainedInputs() CompileOption {
	return func(opt *compileConfig) error {
		opt.ignoreUnconstrainedInputs = true
		return nil
	}
}

// WithBuilder is a compile option which enables the compiler to build the
// constraint system with a user-defined builder.
//
// /!\ This is highly experimental and may change in upcoming releases /!\
func WithBuilder(builder NewBuilder) CompileOption {
	return func(opt *compileConfig) error {
		opt.newBuilder = builder
		return nil
	}
}
