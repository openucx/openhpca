# Introduction

The 'overlap' benchmark suite aims at evaluating the overlap communication/computation.
Compared to other benchmarks, this suite aims at calculating the maximum work that can be
injected in the context of a MPI collective operation and each benchmarks returns results
based on actual work injected during the operation with the guarantee that it did not
increase the execution time of the MPI operation.

As a result, the suite also gives a practical idea of how much work can be injected during
a collective operation.

## Overlap calculation

To achieve that goal, the benchmark, for supported collective operations, calculate first
the time to execute the operation, from the time it it initiated to the time it completed.
Practically, a warm-up loop is first executed, followed by a timing loop where the mean of
the execution times is used as reference time. This reference time is then used to estimate
the equalivent maximum amount of work. Finally, we execute a loop during which we try to
inject the maximum amount of time. if the time is less than the reference time, it means 
more work could be injected. If the time is more than the reference time, it means too
much work has been injected.
Benchmarks are therefore composed of the following phases:
1. Calculate the mean execution time, also called the reference time, for the target MPI
collective, without injecting any work.
Practically, it means that the operation is invoked and MPI_Wait() is called right after the
call to the MPI collective returns. The standard deviation is also calculated.
2. Calculate the equivalent amoutn of work for the reference time.
3. If the execution time of the collective in addition of the injected work and MPI_Wait() is
greater than the reference time, redo the step after dividing the amount of time in half.
Otherwise increase the amount of work to be injected and move to Step 4.
4. If the execution time is:
  - lesser than the reference time, increase the reference time and redo the step
  - greater than the reference time, run the same configuration for validation (5 times by
    default). If all 5 runs reports that the execution time if greater than the reference time,
    the maximum amount of work that can be injected has been found. If at least one of the
    validation runs reports an execution time lesser than the reference time, increase the
    amount of work to be injected and restart at Step 4 with the new amount of work to be
    injected.

## Minimizing result variability

To try to have stable results, the reference time is calculated as previously presented, as
well as the standard deviation. When work is injected, instead we use the mean time plus the
standard deviation to evaluate whether the injected work can be executed during the MPI
collective.

In addition to this mechanism, when the benchmark finds a configuration where the injected work
increases the overall execution time, instead of flagging the configuration has injecting too
much work, beyond what a perfect overlap would allow, the same configuration is executed
multiple times. If a single run reports that the injected work can be executed during the 
collective, it assumed that the amount of work injected can be injected and the benchmark then
tries to inject more work. This allows us to find the maximum amount of wrk that can injected.

# Installation

Update your environment to ensure that the MPI installation you wish to use is available in
PATH and LD_LIBRARY_PATH. For instance, based on your environment, this may look like:
```
export PATH=/path/to/mpi/bin:$PATH
export LD_LIBRARY_PATH=/path/to/mpi/lib:$LD_LIBRARY_PATH
```
Then, run `make`.

# Running and tuning

The overlap benchmark has two different execution models. The first execution models is the one usually adopted by MPI benchmarks, i.e.,
the overlap is calculated based on the data size to be exchanged. We call that model `data driven execution`. The second execution model is time based,
meaning the benchmark finds how long a given test should run, and as a result how much data should be exchanged, and calculate the
overlap based on that time. We call that model `time driven execution`.

The two models have their benefits and drawbacks. The data driven execution model, based on the specific data size, may not offer a result that is
statistically relevant. Basically, if the execution time is too small, and based on the fact that all non-blocking MPI operations see some
variability between runs, the computed overlap may not be representative of any potential application-level opportunity for overlap.
Finally, the data driven execution model is also running the overlap test for many datatype sizes, potentially leading to long execution
time based on the implemented approach.

The time driven execution model aims at identifying statistically relevant results without increasing the execution time.
When running under that model, the test ensures to find a data size that lead to an execution time that is big enough to
enable the calculation of overlap but not too big to lead to a long execution (in the order of a few seconds).
That time is dependent on the scale both in terms of ranks and number of nodes.
Finally, since the time driven execution time is not based on the data size, it enables exchanging different sizes in MPI
collective such as `MPI_Ialltoallv`, where each rank can send/receive a different amount of data.

The time driven model is the default, except for `MPI\_Ibarrier`, since it does not provide any parameter that can be used
to control the execution time (only the number of ranks impacts the execution time, which is not under the control of our benchmark).

## Environment variables

The following environment variables are available to control and tune the execution of the
benchmarks:
- `OPENHPCA_OVERLAP_MIN_NUM_ELTS`, which enables the modification of the minimum number of elements to use (default: 1). This number of elements is multiplied by the datatype size to set the minimum data size that needs to be exchanged for the overlap evaluation of a given MPI collective operation.
- `OPENHPCA_OVERLAP_MAX_NUM_ELTS`, which enables the modification of the maximum number of elements to use (default: 131072). This number of elements is multiplied by the datatype size to set the maximum data size that needs to be exchanged for the overlap evaluation of a given MPI collective operation.
- `OPENHPCA_OVERLAP_VALIDATION_STEPS`, which enables the modifications of the number of validation steps while calculating the overlap (default: 2).
- `OPENHPCA_OVERLAP_VERBOSE`, which enables the verbose mode.
- `OPENHPCA_OVERLAP_DEBUG`, which enables the debug mode.
- `OPENHPCA_DATA_DRIVEN_EXECUTION`, which disables the time driven model and enables the data driven model.
- `OPENHPCA_OVERLAP_CUTOFF_TIME`, which is the time (in milliseconds) used under the time driven model to specify the minimum execution time of the collective operation; large enough to lead to statistically relevant results.
- `OPENHPCA_OVERLAP_ACCEPTANCE_THRESHOLD`, which is the percentage between an amount of injected work that can be overlaped and the known amount of injected work that does not allow perfect overlap that stops the test for the final overlap calculation.
- `OVERLAP_DEFAULT_TDM_N_ITERS_ENVVAR`, which is the default of iterations to execute a MPI collective operation during benchmarking.
- `OVERLAP_MAX_TDM_ITERS_ENVVAR`, which is the maximum number of iterations to use during benchmarking.