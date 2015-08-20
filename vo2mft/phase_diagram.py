from argparse import ArgumentParser
from copy import deepcopy
import numpy as np
import matplotlib.pyplot as plt
from vo2mft.environment import QJ_ion
from vo2mft.min_free_energy import minimize_free_energy
from vo2mft.solve import read_env_file

def phase_sample(base_env, num_Bs, num_Ts):
    '''Generate a set of envs over which the phase diagram will be sampled,
    with constant properties taken from base_env.
    '''
    # Generate set of (B, T) values.
    # TODO - generalize away from EpsilonM - EpsilonR = 0.
    base_QJ_ion = QJ_ion(base_env)
    Bratio_start, Bratio_stop = 0.01, 0.75
    Tratio_start, Tratio_stop = 0.1, 1.0

    Bratios = np.linspace(Bratio_start, Bratio_stop, num_Bs)
    Tratios = np.linspace(Tratio_start, Tratio_stop, num_Ts)

    B_T_vals = []
    for Br in Bratios:
        for Tr in Tratios:
            B_val = Br*base_QJ_ion
            T_val = Tr*base_QJ_ion
            B_T_vals.append([B_val, T_val])

    sample_envs = []
    for B, T in B_T_vals:
        this_env = deepcopy(base_env)
        this_env["B"] = B
        this_env["Beta"] = 1.0/T
        sample_envs.append(this_env)

    return sample_envs

def min_envs_from_sample(sample_envs):
    eps = 1e-6
    min_envs = []
    for env in sample_envs:
        min_env = minimize_free_energy(env, eps)
        min_envs.append(min_env)

    return min_envs

def min_envs_from_base(base_path, num_Bs, num_Ts):
    base_env = read_env_file(base_path)
    sample = phase_sample(base_env, num_Bs, num_Ts)
    min_envs = min_envs_from_sample(sample)
    return min_envs

def _collect_BTM(min_envs):
    xs, ys, Ms = [], [], []
    for this_env in min_envs:
        # May not have found a solution for all envs.
        if this_env == None:
            continue
        # This env was solved -- add it to plot.
        this_QJ_ion = QJ_ion(this_env)
        Bratio = this_env["B"] / this_QJ_ion
        Tratio = (1.0 / this_env["Beta"]) / this_QJ_ion
        xs.append(Bratio)
        ys.append(Tratio)
        Ms.append(this_env["M"])

    return xs, ys, Ms

def _make_M_diagram(Bs, Ts, Ms, out_prefix):
    plt.scatter(Bs, Ts, c=Ms, cmap='gnuplot', s=100, edgecolors="none")
    plt.colorbar()

    if out_prefix == None:
        plt.show()
    else:
        plt.savefig(out_prefix + '.png', bbox_inches='tight', dpi=500)

    plt.clf()

def _main():
    parser = ArgumentParser(description="Construct phase diagram")
    parser.add_argument('--base_env_path', type=str, help="Base environment file path",
            default="phase_diagram_env.json")
    parser.add_argument('--out_prefix', type=str, help="Output file path prefix",
            default="out_phase_diagram")
    parser.add_argument('--num_Ts', type=int, help="Number of temperature points",
            default=20)
    parser.add_argument('--num_Bs', type=int, help="Number of B points",
            default=20)
    args = parser.parse_args()

    # TODO - don't assume run in own directory
    min_envs = min_envs_from_base(args.base_env_path, args.num_Ts, args.num_Bs)

    Bs, Ts, Ms = _collect_BTM(min_envs)
    _make_M_diagram(Bs, Ts, Ms, args.out_prefix)

if __name__ == "__main__":
    _main()