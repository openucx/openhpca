#!/bin/bash -l
#
#SBATCH -p defq
#SBATCH -N 4
#SBATCH -t 1:00:00
#SBATCH --error=6ad2364c6e2c98369551dbcb2f80d39a3e82e759112cb7e2fae49999-220227152833-openmpi4.1.0.err
#SBATCH --output=6ad2364c6e2c98369551dbcb2f80d39a3e82e759112cb7e2fae49999-220227152833-openmpi4.1.0.out


MPI_DIR=/cm/shared/apps/openmpi/gcc/64/4.1.0-ucx
export PATH=$MPI_DIR/bin:$PATH
export LD_LIBRARY_PATH=$MPI_DIR/lib:$LD_LIBRARY_PATH


which mpirun

mpirun -np 4 --map-by ppr:1:node -rank-by core -bind-to core --mca btl ^openib --mca pml ucx /home/bwilliams/openhpca/build/install/overlap/overlap/overlap_ialltoall
