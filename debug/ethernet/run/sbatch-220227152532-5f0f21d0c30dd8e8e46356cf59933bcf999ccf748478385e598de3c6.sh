#!/bin/bash -l
#
#SBATCH -p defq
#SBATCH -N 2
#SBATCH -t 1:00:00
#SBATCH --error=5f0f21d0c30dd8e8e46356cf59933bcf999ccf748478385e598de3c6-220227152532-openmpi4.1.0.err
#SBATCH --output=5f0f21d0c30dd8e8e46356cf59933bcf999ccf748478385e598de3c6-220227152532-openmpi4.1.0.out


MPI_DIR=/cm/shared/apps/openmpi/gcc/64/4.1.0-ucx
export PATH=$MPI_DIR/bin:$PATH
export LD_LIBRARY_PATH=$MPI_DIR/lib:$LD_LIBRARY_PATH


which mpirun

mpirun -np 2 --map-by ppr:1:node -rank-by core -bind-to core --mca btl ^openib --mca pml ucx /home/bwilliams/openhpca/build/install/OSU/libexec/osu-micro-benchmarks/mpi/pt2pt/osu_latency
