package vo2solve

import (
	"math"
)
import (
	"github.com/tflovorn/scExplorer/solve"
	vec "github.com/tflovorn/scExplorer/vector"
)

// Return the absolute error and gradient of the M equation w.r.t. the given
// variables (which should be fixed to ["M", "W", "Mu"] for this case).
func AbsErrorM(env *Environment, Ds *HoppingEV, variables []string) solve.Diffable {
	F := func(v vec.Vector) (float64, error) {
		env.Set(v, variables)
		exp := math.Exp(-env.Beta * (env.DeltaS() - env.W*env.QK()))
		lhs := env.M
		rhs := 2.0 * exp * math.Sinh(env.Beta*env.M*env.QJ(Ds)) / env.Z1(Ds)
		return lhs - rhs, nil
	}
	h := 1e-6
	epsabs := 1e-4
	return solve.SimpleDiffable(F, len(variables), h, epsabs)
}
