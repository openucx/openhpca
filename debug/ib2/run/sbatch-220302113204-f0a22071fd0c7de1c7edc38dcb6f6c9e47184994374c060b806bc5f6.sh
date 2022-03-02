#!/bin/bash -l
#
#SBATCH -p thor
#SBATCH -N 2
#SBATCH -t 1:00:00
#SBATCH --error=f0a22071fd0c7de1c7edc38dcb6f6c9e47184994374c060b806bc5f6-220302113204-openmpi4.1.1.err
#SBATCH --output=f0a22071fd0c7de1c7edc38dcb6f6c9e47184994374c060b806bc5f6-220302113204-openmpi4.1.1.out


MPI_DIR=/global/home/users/bwilliams/ompi_x86/build
export PATH=$MPI_DIR/bin:$PATH
export LD_LIBRARY_PATH=$MPI_DIR/lib:$LD_LIBRARY_PATH


which mpirun

mpirun -np 2 --map-by ppr:1:node -rank-by core -bind-to core --mca btl ^openib --mca pml ucx /global/home/users/bwilliams/openhpca/build/install/mpi_overhead/mpi_overhead/mpi_overhead
