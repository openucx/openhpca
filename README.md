# openhpca

OpenHPCA is an implementation of the benchmarks defined by the High Performance
Compute Availability (HPCA) group.
HPCA aims at providing a comprehensive set of benchmarks to evaluate the
overall compute resource performance in the presence of in-network computing
technologies. These benchmarks are a mix of existing benchmarks as well as new
benchmarks that are defined and implemented by the group.

The benchmarks included in OpenHPCA are:
- Sandia micro-benchmarks (SMB): https://cs.sandia.gov/smb/
- the OSU micro-benchmarks: http://mvapich.cse.ohio-state.edu/benchmarks/
- a modified version of the OSU micro-benchmarks for non-contiguous data
- a OpenHPCA benchmark suite evaluating the overlap capabilities in the
context of non-blocking MPI operations (overlap).

OpenHPCA currently relies on the following versions of the different external
benchmarks:
- OSU 5.7
- SMB from `https://github.com/sandialabs/SMB`

# Installation

Since OpenHPCA is composed of already existing benchmarks and new benchmarks,
an entire infrastructure has been developed to integrate them together and
make it easier to install and run. While users are encouraged to use the
integrated infrastructure, it is also possible to manually install all the
different benchmarks.

## Pre-requirements

The OpenHPCA software requires the following components to be installed prior
to setting it up:
- Go, version 1.14 or newer,
- an MPI implementation.

## Using the integrated infrastructure

OpenHPCA relies on a workspace so before installing OpenHPCA for the first
time, users
are required to define the configuration of the workspace. A new workspace can
be specified by creating the `~/.openhpca/workspace.conf` file. Only one
workspace is supported at any given time at the moment, multiple workspaces
can only be currently used by creating a symlink to the target configuration
file that needs to be used at a given time. Future versions of the suite will
support multiple workspace so multiple MPI installation could be evaluated
in parallel. The content of the file should look like:

```
dir = <PATH/TO/DIRECTORY>
mpi = <PATH/TO/THE/MPI/INSTALLATION/TO/USE/WITH/OPENHPCA>
```
Only two configuration parameters are required:
- `dir`, which specifies where the workspace will be practically deployed, i.e.,
where all the data, source codes, compiled data is stored. Note that it must be
on a location that is accessible from compute nodes when installed on a cluster.
- `mpi`, which points to the MPI installation to use to run the MPI benchmarks
(e.g., SMB, OSU, overlap).
Once the workspace configuration created, execute the following command:
    `make init`

Once the workspace defined, users can install the benchmarks simply by running
`make install`. This configures, builds and installs all the benchmarks
in the workspace.

## Manual installation

### OSU

While the integrated infrastructure will automatically download and install the
OSU micro-benchmarks, a manual installation requires users to manually download,
configure and install it. Please refer to the OSU documentation for details.

### SMB

While the integrated infrastructure will automatically download and install the
SMB micro-benchmarks, a manual installation requires users to manually download,
configure and install it. Please refer to the documentation available from the
Github project: https://github.com/sandialabs/SMB.

### overlap

Please refer to the `src/overlap/README.md` file for instructions.

# Execution

## Using the integrated framework

OpenHPCA provides the `openhpca_run` tool that automatically run all the
benchmarks following best practices for such benchmarks. This tool can
interface with Slurm and other job management systems. Note that the
current implementation focuses on Slurm and already provides some support
for SSH based configuration. If support for other job managers is required,
please contact the development team. The infrastructure has been designed
to be easily extensible and support for additional job managers should be
doable with minimum effort.

To run the benchmarks, the full command line looks like:
```
./tools/cmd/openhpca_run/openhpca_run -d mlx5_0:1 -p cluster_partition
```
Where `mlx5_0:1` is the device to use for the execution of the benchmarks and
`cluster_partition` the partition to use, for instance, on a Slurm cluster.
For a full description of the supported parameters, please execute
`./tools/cmd/openhpca_run/openhpca_run -h` from the top directory of the
OpenHPCA source code.

## Manual execution

For a manual execution, users are asked to run the various benchmarks as they
would usually do. All the benchmarks are available from the `install` directory
in the workspace that they defined.

# Data visualization

Since OpenHPCA generates a fairly large of data, the recommended way to
visualize all the results is to use the OpenHPCA viewer.

To start the viewer, simply execute the following command from the top
directory of the OpenHPCA source code:
```
./tools/cmd/webui/webui -port 8082
```
In this example, the viewer starts on the 8082 port (8080 is used by default).

For details about all the supported parameters, please refer to the
`./tools/cmd/webui/webui -h` command output.

As a note, to connect to a remote server through SSH where the webui is meant
to be executed, use a SSH port forwarding command such as:
```
ssh -L 8080:127.0.0.1:8080 remote-server
```
You can the connect to the remote server and access the webui using your
preferred web browser on your local machine by using the following URL:
```
http://127.0.0.1:8080
```

# Project governance

## Versioning

The version of the OpenHPCA software follows the X.Y.Z scheme.

The addition of any new metrics reported to users necessitates to increment X.
The addition of any new feature without any new metrics necessitates to increase Y.
Bug fixes to the existing code necessitates to increase Z.

## Release process

Once the group agrees on initiating the release process, a first release candidate (RC) is created. If no issue related to the upcoming release is open between two group virtual meetings, the release can be finalized. Otherwise, pull requests are reviewed during the virtual meetings and a new RC created and made public at the end of each meeting.
