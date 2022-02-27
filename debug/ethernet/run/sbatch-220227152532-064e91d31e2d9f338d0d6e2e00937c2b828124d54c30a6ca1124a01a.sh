#!/bin/bash -l
#
#SBATCH -p defq
#SBATCH -N 4
#SBATCH -t 1:00:00
#SBATCH --error=064e91d31e2d9f338d0d6e2e00937c2b828124d54c30a6ca1124a01a-220227152532-openmpi4.1.0.err
#SBATCH --output=064e91d31e2d9f338d0d6e2e00937c2b828124d54c30a6ca1124a01a-220227152532-openmpi4.1.0.out


MPI_DIR=/cm/shared/apps/openmpi/gcc/64/4.1.0-ucx
export PATH=$MPI_DIR/bin:$PATH
export LD_LIBRARY_PATH=$MPI_DIR/lib:$LD_LIBRARY_PATH


which mpirun

mpirun -np 4 --map-by ppr:1:node -rank-by core -bind-to core --mca btl ^openib --mca pml ucx /home/bwilliams/openhpca/build/install/overlap/overlap/overlap_iallreduce
