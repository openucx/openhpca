#!/bin/bash -l
#
#SBATCH -p thor
#SBATCH -N 2
#SBATCH -t 1:00:00
#SBATCH --error=f19b68df31f6679e42668ee3f7c2ef326fa026c8ec7061986bca77ac-220226145427-openmpi4.1.1.err
#SBATCH --output=f19b68df31f6679e42668ee3f7c2ef326fa026c8ec7061986bca77ac-220226145427-openmpi4.1.1.out


MPI_DIR=/global/home/users/bwilliams/ompi_x86/build
export PATH=$MPI_DIR/bin:$PATH
export LD_LIBRARY_PATH=$MPI_DIR/lib:$LD_LIBRARY_PATH


which mpirun

mpirun -np 2 --map-by ppr:1:node -rank-by core -bind-to core --mca btl ^openib --mca pml ucx -x UCX_NET_DEVICES=mlx5_0:1 /global/home/users/bwilliams/openhpca/build/install/mpi_overhead/mpi_overhead/mpi_overhead
