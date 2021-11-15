//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

#include <stdlib.h>
#include <stdint.h>
#include <unistd.h>

#include "mpi.h"
#include "overlap.h"
#include "overlap_ddm.h"

#define FREEMEM             \
    do                      \
    {                       \
        FINI_OVERLAP_BENCH; \
    } while (0)

volatile double x = 1.0, y = 1.0, a = 1.0, b = 1.0;

int data_driven_loop(overlap_params_t *params)
{
    DDM_VARIABLES
    INIT_OVERLAP_LOOP

    // Iterate over data size
    for (n_elts = params->min_elts; n_elts <= params->max_elts; n_elts *= 2)
    {
        INIT_OVERLAP_STATUS(params, (&overlap_status));

        /* Get reference numbers */
        if (params->world_rank == 0)
            OVERLAP_DEBUG(params, "Getting reference data for %ld bytes...\n", n_elts * sizeof(double));
        work_total = 0.0;
        wait_total = 0.0;
        post_total = 0.0;
        total_time = 0.0;
        MPI_Barrier(MPI_COMM_WORLD); // Make sure to sync ranks before moving on, we don't want late arrivals
        // We mimic the loop to gather data so we can make meaning full comparisons
        for (n = 0; n < n_iters; n++)
        {
            start_time = MPI_Wtime();
            MPI_CHECK(MPI_Ibarrier(MPI_COMM_WORLD, &req));
            end_post = MPI_Wtime();
            start_work = MPI_Wtime();
            asm volatile("nop");
            end_work = MPI_Wtime();
            start_wait = MPI_Wtime();
            MPI_CHECK(MPI_Wait(&req, &status));
            end_time = MPI_Wtime();
            total_time += (end_post - start_time) + (end_work - start_work) + (end_time - start_wait);
            ref_data[n] = (end_post - start_time) + (end_work - start_work) + (end_time - start_wait);
            work_times[n] = end_work - start_work;
            wait_times[n] = end_time - start_wait;
            post_times[n] = end_post - start_time;
            work_total += end_work - start_work;
            wait_total += end_time - start_wait;
            post_total += end_post - start_time;
            MPI_Barrier(MPI_COMM_WORLD); // Make sure to sync ranks before moving on, we don't want late arrivals
        }

        COMPUTE_REQUIRED_WORK;

        while (work > 0)
        {
            total_time = 0.0;
            work_total = 0.0;
            wait_total = 0.0;
            post_total = 0.0;
            // Warm up
            for (n = 0; n < warmup; n++)
            {
                start_time = MPI_Wtime();
                MPI_CHECK(MPI_Ibarrier(MPI_COMM_WORLD, &req));
                end_post = MPI_Wtime();
                start_work = MPI_Wtime();
                do_work(x, y, a, b, work);
                end_work = MPI_Wtime();
                start_wait = MPI_Wtime();
                MPI_CHECK(MPI_Wait(&req, &status));
                end_time = MPI_Wtime();
                MPI_Barrier(MPI_COMM_WORLD); // Make sure to sync ranks before moving on, we don't want late arrivals
            }

            // Actual benchmarking loop
            for (n = 0; n < n_iters; n++)
            {
                start_time = MPI_Wtime();
                MPI_CHECK(MPI_Ibarrier(MPI_COMM_WORLD, &req));
                end_post = MPI_Wtime();
                start_work = MPI_Wtime();
                do_work(x, y, a, b, work);
                end_work = MPI_Wtime();
                start_wait = MPI_Wtime();
                MPI_CHECK(MPI_Wait(&req, &status));
                end_time = MPI_Wtime();
                total_time += (end_post - start_time) + (end_work - start_work) + (end_time - start_wait);
                data[n] = (end_post - start_time) + (end_work - start_work) + (end_time - start_wait); // not used but make sure we always the same memory accesses than for the computation of the reference times
                work_times[n] = end_work - start_work;
                wait_times[n] = end_time - start_wait;
                post_times[n] = end_post - start_time;
                work_total += end_work - start_work;
                wait_total += end_time - start_wait;
                post_total += end_post - start_time;
                MPI_Barrier(MPI_COMM_WORLD); // Make sure to sync ranks before moving on, we don't want late arrivals
            }

            DDM_GATHER_AND_PROCESS_DATA;
            MPI_CHECK(MPI_Bcast(&work, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD));
            MPI_CHECK(MPI_Barrier(MPI_COMM_WORLD));
        }
    }

    FREEMEM;
    return 0;

exit_error:
    FREEMEM;
    MPI_Abort(MPI_COMM_WORLD, 1);
    return 1;
}

int main(int argc, char **argv)
{
    INIT_OVERLAP_BENCH;
    rc = data_driven_loop(&params);
    if (rc)
    {
        fprintf(stderr, "Benchmark function failed (return code = %d)\n", rc);
        goto exit_error;
    }

    MPI_Finalize();
    return (EXIT_SUCCESS);

exit_error:
    MPI_Abort(MPI_COMM_WORLD, 1);
    return (EXIT_FAILURE);
}