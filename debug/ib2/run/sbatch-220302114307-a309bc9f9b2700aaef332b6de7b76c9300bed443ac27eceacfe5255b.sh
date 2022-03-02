#!/bin/bash -l
#
#SBATCH -p thor
#SBATCH -N 2
#SBATCH -t 1:00:00
#SBATCH --error=a309bc9f9b2700aaef332b6de7b76c9300bed443ac27eceacfe5255b-220302114307-openmpi4.1.1.err
#SBATCH --output=a309bc9f9b2700aaef332b6de7b76c9300bed443ac27eceacfe5255b-220302114307-openmpi4.1.1.out


MPI_DIR=/global/home/users/bwilliams/ompi_x86/build
export PATH=$MPI_DIR/bin:$PATH
export LD_LIBRARY_PATH=$MPI_DIR/lib:$LD_LIBRARY_PATH


which mpirun

mpirun -np 2 --map-by ppr:1:node -rank-by core -bind-to core --mca btl ^openib --mca pml ucx /global/home/users/bwilliams/openhpca/build/install/OSU/libexec/osu-micro-benchmarks/mpi/pt2pt/osu_bw
