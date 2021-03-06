from copy import deepcopy
from vo2mft.solve import solve_set

def minimize_free_energy(env, eps=1e-6, ions=False, twodof=False, twodof_body_indep=False, initial_vals=None):
    '''Vary the initial conditions of env to find the solution which
    minimizes the free energy. Return the minimum free energy env and
    a list of all the solved envs.
    '''
    # Set of initial conditions to consider.
    # TODO - may need to expand this.
    initial_conds, flags = None, None
    if not twodof:
        initial_conds = [{"M": 0.01, "W":0.01}, {"M":1.0, "W":1.0}]
    elif twodof and twodof_body_indep:
        initial_conds = [{"M01": 1.0, "M11": 1.0, "M02": 1.0, "M12": 1.0, "W01": 1.0, "W11": 1.0, "W02": 1.0, "W12": 1.0},
                {"M01": 0.01, "M11": 0.01, "M02": 0.01, "M12": 0.01, "W01": 0.7, "W11": 0.7, "W02": 0.7, "W12": 0.7},
                {"M01": 1.0, "M11": 1.0, "M02": 0.01, "M12": 0.01, "W01": 1.0, "W11": 1.0, "W02": 0.7, "W12": 0.7},
                {"M01": 1.0, "M11": 1.0, "M02": 1.0, "M12": 0.01, "W01": 1.0, "W11": 1.0, "W02": 1.0, "W12": 0.7},
                {"M01": 1.0, "M11": 1.0, "M02": 0.01, "M12": 1.0, "W01": 1.0, "W11": 1.0, "W02": 0.7, "W12": 1.0}]
        #flags = [[], ["--m01_0", "--m11_0", "--m02_0", "--m12_0"],
        #        ["--m02_0", "--m12_0"], ["--m12_0"], ["--m02_0"]]
        flags = [[], [], [], [], [], []]
        if initial_vals == "mode_symmetric":
            # Consider M2 with both mode 1 -> 0 and mode 2 -> 0.
            # Appropriate when the Hamiltonian breaks the symmetry between these.
            # If the Hamiltonian is symmetric, want to break the symmetry
            # in initial conditions to avoid solution fluctuating between these.
            initial_conds.append({"M01": 0.01, "M11": 0.01, "M02": 1.0, "M12": 1.0, "W01": 0.7, "W11": 0.7, "W02": 1.0, "W12": 1.0})
            flags.append([])

    else:
        initial_conds = [{"M01": 1.0, "M02": 1.0}, {"M01": 1.0, "M02": 1.0},
                {"M01": 1.0, "M02": 1.0}]
        flags = [[], ["--m01_0", "--m02_0"], ["--m02_0"]]

    # Set up envs with specified set of initial conditions.
    initial_envs = []
    for cond in initial_conds:
        this_initial_env = deepcopy(env)
        for k, v in cond.items():
            this_initial_env[k] = v
        initial_envs.append(this_initial_env)

    # Solve envs.
    final_envs = solve_set(initial_envs, eps, ions, flags, twodof, twodof_body_indep)
    #print(final_envs)

    # Find env with minimum free energy.
    min_env = None
    for final_env in final_envs:
        # May not have found a solution.
        if final_env == None:
            continue
        free_energy = final_env["FreeEnergy"]
        if min_env == None or free_energy < min_env["FreeEnergy"]:
            min_env = final_env

    return min_env, final_envs
