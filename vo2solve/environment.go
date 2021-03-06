package vo2solve

import (
	"fmt"
	"io/ioutil"
	"math"
	"reflect"
)
import (
	"github.com/tflovorn/cmatrix"
	"github.com/tflovorn/scExplorer/bzone"
	"github.com/tflovorn/scExplorer/serialize"
	vec "github.com/tflovorn/scExplorer/vector"
)

// Contains parameters necessary to characterize electronic and ionic systems.
// The ionic order parameters M and W and the electronic chemical potential Mu
// must be determined self-consistently.
type Environment struct {
	// Size of BZ on one edge (total number of BZ points is this cubed).
	BZPointsPerDim int
	// Hopping parameters, even symmetry (a, c, diagonal axes).
	Tae, Tce, Tbe float64
	// Hopping parameters, odd symmetry (a, c, diagonal axes).
	Tao, Tco, Tbo float64
	// Order parameter <S>.
	M float64
	// Order parameter <S^2>.
	W float64
	// Chemical potential.
	Mu float64
	// Inverse temperature, 1 / (k_B * T).
	Beta float64
	// One-spin term for BEG model: coefficient for (S_i)^2.
	B float64
	// Exchange parameters for BEG model: coefficients to S_i dot S_j.
	// Jb is excluded since it does not contribute to results.
	Ja, Jc float64
	// Biquadratic exchange parameters for BEG model: coefficients to (S_i)^2 * (S_j)^2.
	Ka, Kc, Kb float64
	// On-site energies in M and R phases.
	EpsilonM, EpsilonR float64
	// Consider only ionic part of the problem:
	// only ions contribute to free energy; should solve
	// for (M, W).
	// If this is set to true, need to also set the following to 0:
	// Tae, Tce, Tbe, Tao, Tco, Tbo, EpsilonM, EpsilonR, Mu.
	// (maybe don't need to fix Mu = 0 -- large negative value could
	// be better).
	IonsOnly bool
}

// Environment with all self-consistent values converged.
// Includes additional data for exporting to outside programs.
type FinalEnvironment struct {
	Environment
	Dae, Dce, Dbe, Dao, Dco, Dbo float64
	FreeEnergy                   float64
}

func (env *Environment) DeltaS() float64 {
	return env.B + env.EpsilonM - env.EpsilonR
}

// Combined biquadratic coefficient (S_i^2 S_j^2).
func (env *Environment) QK() float64 {
	return 4.0*env.Ka + 2.0*env.Kc + 8.0*env.Kb
}

// Combined renormalized 'exchange' coefficient (S_i S_j) favoring dimers.
func (env *Environment) QJ(Ds *HoppingEV) float64 {
	Dao, Dco := Ds.Dao(env), Ds.Dco(env)
	return 4.0*(env.Ja+env.Tao*Dao) + 2.0*(env.Jc+env.Tco*Dco)
}

func (env *Environment) Qele(Ds *HoppingEV) float64 {
	// TODO - make sure T's here should be even part.
	Dae, Dce, Dbe := Ds.Dae(env), Ds.Dce(env), Ds.Dbe(env)
	return 4.0*env.Tae*Dae + 2.0*env.Tce*Dce + 8.0*env.Tbe*Dbe
}

func (env *Environment) Z1(Ds *HoppingEV) float64 {
	exp := math.Exp(-env.Beta * (env.DeltaS() - env.W*env.QK()))
	return 1.0 + 2.0*exp*math.Cosh(env.Beta*env.M*env.QJ(Ds))
}

// Are electronic hopping finite?
// If not, don't need to calculate D's.
func (env *Environment) FiniteHoppings() bool {
	eps := 1e-9
	even := (math.Abs(env.Tae) > eps) || (math.Abs(env.Tce) > eps) || (math.Abs(env.Tbe) > eps)
	odd := (math.Abs(env.Tao) > eps) || (math.Abs(env.Tco) > eps) || (math.Abs(env.Tbo) > eps)
	return even || odd
}

// Fermi distribution function.
func (env *Environment) Fermi(energy float64) float64 {
	// Need to make this check to be sure we're dividing by a nonzero energy in the next step.
	if energy == 0.0 {
		return 0.5
	}
	// Temperature is 0 or e^(Beta*energy) is too big to calculate
	if env.Beta == math.Inf(1) || env.Beta >= math.Abs(math.MaxFloat64/energy) || math.Abs(env.Beta*energy) >= math.Log(math.MaxFloat64) {
		if energy <= 0 {
			return 1.0
		}
		return 0.0
	}
	// nonzero temperature
	return 1.0 / (math.Exp(energy*env.Beta) + 1.0)
}

// Free energy per cell value (Ncell = 2Nsite).
// Points on the phase diagram include the state with minimum free energy
// (may not reach this state, depending on initial conditions - need to
// consider a set of initial conditions and look for minimum).
func (env *Environment) FreeEnergy(Ds *HoppingEV) float64 {
	ion_part := env.FreeEnergyIons(Ds)
	// avg_avg_part includes <S><S>, <S^2><S^2>, and <S><c^{\dagger}c> terms.
	avg_avg_part := env.QJ(Ds)*math.Pow(env.M, 2.0) + env.QK()*math.Pow(env.W, 2.0) + env.Qele(Ds)
	if env.IonsOnly {
		return ion_part + avg_avg_part
	} else {
		electron_part := env.FreeEnergyElectrons()
		return ion_part + electron_part + avg_avg_part
	}
}

func (env *Environment) FreeEnergyIons(Ds *HoppingEV) float64 {
	T := 1.0 / env.Beta
	return -2.0 * T * math.Log(env.Z1(Ds))
}

func (env *Environment) FreeEnergyElectrons() float64 {
	inner := func(k vec.Vector) float64 {
		H := ElHamiltonian(env, k)
		dim, _ := H.Dims()
		evals, _ := cmatrix.Eigensystem(H)
		sum := 0.0
		for alpha := 0; alpha < dim; alpha++ {
			eps_ka := evals[alpha]
			// Mu excluded from exp argument here since it is
			// included in H.
			val := 1.0 + math.Exp(-env.Beta*eps_ka)
			sum += 2.0 * math.Log(val)
		}
		return sum
	}
	L := env.BZPointsPerDim
	T := 1.0 / env.Beta
	band_part := -T * bzone.Avg(L, 3, inner)
	n := 1.0
	mu_part := 2.0 * env.Mu * n

	return band_part + mu_part
}

// Create an Environment from the given serialized data.
func NewEnvironment(jsonData string) (*Environment, error) {
	// initialize env with input data
	env := new(Environment)
	err := serialize.CopyFromJSON(jsonData, env)
	if err != nil {
		return nil, err
	}

	return env, nil
}

// Create a FinalEnvironment from the given solved Environment and associated
// HoppingEV.
func NewFinalEnvironment(env *Environment, Ds *HoppingEV) *FinalEnvironment {
	Dae := Ds.Dae(env)
	Dce := Ds.Dce(env)
	Dbe := Ds.Dbe(env)
	Dao := Ds.Dao(env)
	Dco := Ds.Dco(env)
	Dbo := Ds.Dbo(env)
	FreeEnergy := env.FreeEnergy(Ds)
	fenv := FinalEnvironment{*env, Dae, Dce, Dbe, Dao, Dco, Dbo, FreeEnergy}
	return &fenv
}

// Load an Environment from the JSON file at envFilePath.
func LoadEnv(envFilePath string) (*Environment, error) {
	data, err := ioutil.ReadFile(envFilePath)
	if err != nil {
		return nil, err
	}
	env, err := NewEnvironment(string(data))
	if err != nil {
		return nil, err
	}
	return env, nil
}

// Load an Environment from the JSON file at envFilePath.
// Set all electronic parameters to 0 to restrict to ionic system.
func LoadIonEnv(envFilePath string) (*Environment, error) {
	env, err := LoadEnv(envFilePath)
	if err != nil {
		return nil, err
	}
	env.Tae = 0.0
	env.Tce = 0.0
	env.Tbe = 0.0
	env.Tao = 0.0
	env.Tco = 0.0
	env.Tbo = 0.0
	env.EpsilonM = 0.0
	env.EpsilonR = 0.0
	env.Mu = 0.0
	env.IonsOnly = true
	return env, nil
}

// Convert to string by marshalling to JSON
func (env *Environment) String() string {
	marshalled := env.Marshal()
	return marshalled
}

func (env *Environment) Marshal() string {
	if env.Beta == math.Inf(1) {
		// hack to get around JSON's choice to not allow Inf
		env.Beta = math.MaxFloat64
	}
	marshalled, err := serialize.MakeJSON(env)
	if err != nil {
		panic(err)
	}
	if env.Beta == math.MaxFloat64 {
		env.Beta = math.Inf(1)
	}
	return marshalled
}

func (env *FinalEnvironment) String() string {
	marshalled := env.Marshal()
	return marshalled
}

func (env *FinalEnvironment) Marshal() string {
	if env.Beta == math.Inf(1) {
		// hack to get around JSON's choice to not allow Inf
		env.Beta = math.MaxFloat64
	}
	marshalled, err := serialize.MakeJSON(env)
	if err != nil {
		panic(err)
	}
	if env.Beta == math.MaxFloat64 {
		env.Beta = math.Inf(1)
	}
	return marshalled
}

// Iterate through v and vars simultaneously. vars specifies the names of
// fields to change in env (they are set to the values given in v).
// Panics if vars specifies a field not contained in env (or a field of
// non-float type).
func (env *Environment) Set(v vec.Vector, vars []string) {
	ev := reflect.ValueOf(env).Elem()
	for i := 0; i < len(vars); i++ {
		field := ev.FieldByName(vars[i])
		if field == reflect.Zero(reflect.TypeOf(env)) {
			panic(fmt.Sprintf("Field %v not present in Environment", vars[i]))
		}
		if field.Type().Kind() != reflect.Float64 {
			panic(fmt.Sprintf("Field %v is non-float", vars[i]))
		}
		field.SetFloat(v[i])
	}
}
