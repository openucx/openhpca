#!/bin/bash -l
#
#SBATCH -p thor
#SBATCH -N 4
#SBATCH -t 1:00:00
#SBATCH --error=1665dc94dae99dd8601496660be556049bf1e8df48ec92030cc391b0-220302114106-openmpi4.1.1.err
#SBATCH --output=1665dc94dae99dd8601496660be556049bf1e8df48ec92030cc391b0-220302114106-openmpi4.1.1.out


MPI_DIR=/global/home/users/bwilliams/ompi_x86/build
export PATH=$MPI_DIR/bin:$PATH
export LD_LIBRARY_PATH=$MPI_DIR/lib:$LD_LIBRARY_PATH


which mpirun

mpirun -np 4 --map-by ppr:1:node -rank-by core -bind-to core --mca btl ^openib --mca pml ucx /global/home/users/bwilliams/openhpca/build/install/overlap/overlap/overlap_igather
