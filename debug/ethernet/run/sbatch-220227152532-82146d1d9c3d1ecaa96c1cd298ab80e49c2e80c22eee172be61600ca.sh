#!/bin/bash -l
#
#SBATCH -p defq
#SBATCH -N 2
#SBATCH -t 1:00:00
#SBATCH --error=82146d1d9c3d1ecaa96c1cd298ab80e49c2e80c22eee172be61600ca-220227152532-openmpi4.1.0.err
#SBATCH --output=82146d1d9c3d1ecaa96c1cd298ab80e49c2e80c22eee172be61600ca-220227152532-openmpi4.1.0.out


MPI_DIR=/cm/shared/apps/openmpi/gcc/64/4.1.0-ucx
export PATH=$MPI_DIR/bin:$PATH
export LD_LIBRARY_PATH=$MPI_DIR/lib:$LD_LIBRARY_PATH


which mpirun

mpirun -np 2 --map-by ppr:1:node -rank-by core -bind-to core --mca btl ^openib --mca pml ucx /home/bwilliams/openhpca/build/install/mpi_overhead/mpi_overhead/mpi_overhead
