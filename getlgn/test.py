def getLgnProbes(lgn_path, obs_path):
    '''
    return the indexes of the lgn probes.
    lgn_path must be a file with a header, representing edges.
    obs_path must be a file with a header, in the first column must be present the probes
    lgn.txt
    -------
    from,to
    node1,node2
    node1,node3
    node2,node4
    obs.txt
    -------
    probe,exp1,exp2,exp3
    node1,12.4,13.9,19.6
    node2,10.5,17.2,13.1
    node3,16.9,12.7,14.8
    '''
    lgn = []

    with open(lgn_path) as f:
        for row in f.readlines()[1:]:
            lgn += [a.strip() for a in row.strip().split(',')]

    lgnNames = set(lgn) # filters repetitions on probe names
    lgnIndices = []

    with open(obs_path) as f:
        rowNumber = 0

        for row in f.readlines()[1:]:
            probeId = row.split(',')[0]

            if probeId in lgnNames: # the row correspond to a LGN node
                lgnIndices.append(rowNumber)

            rowNumber += 1

    if len(lgnIndices) == 0:
        Print("Creating an empty for lgn: {} and experiment: {}".format(lgn_path, obs_path))

    return (lgnIndices, rowNumber)

print(getLgnProbes("/home/boincadm/projects/test/gene_input_chaos/hs/T096662-CRYZ.lgn", "/home/boincadm/projects/test/gene_input_chaos/hs/hgnc_data_mat.csv"))
