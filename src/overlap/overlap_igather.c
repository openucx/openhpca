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
#include "overlap_tdm.h"

#define FREEMEM             \
    do                      \
    {                       \
        FINI_OVERLAP_BENCH; \
        MEMFREE(s_buf);     \
        MEMFREE(r_buf);     \
    } while (0)

volatile double x, y, a, b;

int data_driven_loop(overlap_params_t *params, double *s_buf, double *r_buf)
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
            MPI_CHECK(MPI_Igather(s_buf, n_elts, MPI_DOUBLE,
                                  r_buf, n_elts, MPI_DOUBLE,
                                  0, MPI_COMM_WORLD, &req));
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
                MPI_CHECK(MPI_Igather(s_buf, n_elts, MPI_DOUBLE,
                                      r_buf, n_elts, MPI_DOUBLE,
                                      0, MPI_COMM_WORLD, &req));
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
                MPI_CHECK(MPI_Igather(s_buf, n_elts, MPI_DOUBLE,
                                      r_buf, n_elts, MPI_DOUBLE,
                                      0, MPI_COMM_WORLD, &req));
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

    MPI_Finalize();
    FREEMEM;
    return 0;

exit_error:
    FREEMEM;
    MPI_Abort(MPI_COMM_WORLD, 1);
    return 1;
}

static inline int
get_coll_config_info(overlap_params_t *params, double *s_buf, double *r_buf, double *data, int n_elts, int num_iters, int work, double *op_stdev, double *avg_time)
{
    double stdev, time_sum = 0;
    double work_start_time, end_time;
    double work_end_time; // Not used but minics what done in main benchmark loop
    int i;
    MPI_Request req;
    MPI_Status status;
    for (i = 0; i < num_iters; i++)
    {
        MPI_Barrier(MPI_COMM_WORLD);
        MPI_CHECK(MPI_Igather(s_buf, n_elts, MPI_DOUBLE,
                              r_buf, n_elts, MPI_DOUBLE,
                              0, MPI_COMM_WORLD, &req));
        work_start_time = MPI_Wtime();
        do_work(x, y, a, b, work);
        work_end_time = MPI_Wtime();
        MPI_CHECK(MPI_Wait(&req, &status));
        end_time = MPI_Wtime();
        data[i] = (end_time - work_start_time) * 1000; // In milli-seconds
        time_sum += end_time - work_start_time;
    }
    time_sum *= 1000; // To milliseconds

    STDEV(data, num_iters, stdev);
    *op_stdev = stdev;
    *avg_time = time_sum / num_iters;
    return 0;
exit_error:
    return 1;
}

int time_driven_loop(overlap_params_t *params, double *s_buf, double *r_buf)
{
    double avg_wait_time = 0, work_time, final_work_time;
    int64_t ref_work;
    double *calibration_data;
    TDM_VARIABLES
    INIT_OVERLAP_LOOP
    n_elts = 1;
    n_iters = TDM_DEFAULT_N_ITERS;
    INIT_OVERLAP_STATUS(params, (&overlap_status));
    MEMALLOC(calibration_data, double, MAX_NUM_CALIBRATION_POINTS * sizeof(double));

    // Find the size that gives an execution time close to the cutoff
    do
    {
        get_coll_config_info(params, s_buf, r_buf, data, n_elts, 5, 0, &stdev, &avg_wait_time);
        if (params->world_rank == 0 && avg_wait_time < params->cutoff_time)
        {
            CHECK_N_ELTS(n_elts, params->max_elts);
            n_elts *= 2;
        }

        MPI_CHECK(MPI_Bcast(&avg_wait_time, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD));
        MPI_CHECK(MPI_Bcast(&n_elts, 8, MPI_BYTE, 0, MPI_COMM_WORLD));
    } while (avg_wait_time < params->cutoff_time);

    if (params->world_rank == 0)
        OVERLAP_DEBUG(params, "Will be using %" PRIu64 " elts (time = %f)\n", n_elts, avg_wait_time);

    while (!done)
    {
        double required_iters;
        do
        {
            // We gather some basic data using the default amount of iterations.
            // Based on the resulting execution time and standard deviation, we calculate how much iterations would
            // be necessary to have relevant results. If the number of iterations is within our limit, we use that
            // configuration, otherwise we recursively increase the amount of data.
            get_coll_config_info(params, s_buf, r_buf, data, n_elts, n_iters, 0, &stdev, &avg_wait_time);
            if (params->world_rank == 0)
            {
                // 1.645 is the critical value for a 90% confidence
                required_iters = pow((1.645 * stdev) / (avg_wait_time / 10), 2);
                OVERLAP_DEBUG(params, "Required number of iterations = %.0f (%" PRIu64 " elts)\n", required_iters, n_elts);
                if (required_iters > MAX_NUM_CALIBRATION_POINTS)
                {
                    CHECK_N_ELTS(n_elts, params->max_elts);
                    n_elts *= 2;
                }
                else
                {
                    if (required_iters > n_iters)
                        n_iters = (int)required_iters;
                    if (required_iters > params->max_iters)
                        n_iters = params->max_iters;
                }
            }

            MPI_CHECK(MPI_Bcast(&n_iters, 1, MPI_INT, 0, MPI_COMM_WORLD));
            MPI_CHECK(MPI_Bcast(&n_elts, 8, MPI_BYTE, 0, MPI_COMM_WORLD));
        } while (required_iters > MAX_NUM_CALIBRATION_POINTS);

        if (n_iters > MAX_NUM_CALIBRATION_POINTS)
            MPI_Abort(MPI_COMM_WORLD, 1);
        done = true;
    }

    // Get the reference time and stdev based on the final configuration
    get_coll_config_info(params, s_buf, r_buf, data, n_elts, n_iters, 0, &stdev, &ref_time);
    MEMFREE(calibration_data);
    TDM_SET_ITERS_AND_ELTS

    // Run the benchmark loop
    while (work > 0)
    {
        // Actual benchmarking loop
        if (params->world_rank == 0)
            OVERLAP_DEBUG(params, "Benchmark loop for work = %" PRId64 "\n", work);
        total_time = 0.0;
        work_time = 0.0;
        MPI_Barrier(MPI_COMM_WORLD); // Make sure to sync ranks before moving on, we don't want late arrivals
        for (n = 0; n < n_iters; n++)
        {
            MPI_CHECK(MPI_Igather(s_buf, n_elts, MPI_DOUBLE,
                                  r_buf, n_elts, MPI_DOUBLE,
                                  0, MPI_COMM_WORLD, &req));
            start_work = MPI_Wtime();
            do_work(x, y, a, b, work);
            end_work = MPI_Wtime();
            MPI_CHECK(MPI_Wait(&req, &status));
            end_time = MPI_Wtime();
            total_time += end_time - start_work;
            work_time += end_work - start_work;
            MPI_Barrier(MPI_COMM_WORLD); // Make sure to sync ranks before moving on, we don't want late arrivals
        }
        total_time *= 1000; // To milliseconds
        work_time *= 1000;  // To milliseconds

        TDM_PROCESS_DATA
        MPI_CHECK(MPI_Bcast(&work, 8, MPI_BYTE, 0, MPI_COMM_WORLD));
    }

    TDM_COMPUTE_OVERLAP
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

    // All the variables specific to ialltoall
    double *s_buf = NULL, *r_buf = NULL;
    MEMALLOC(s_buf, double, params.world_size *params.max_elts * sizeof(double));
    MEMALLOC(r_buf, double, params.world_size *params.max_elts * sizeof(double));

    if (params.data_driven_model)
        rc = data_driven_loop(&params, s_buf, r_buf);
    else
        rc = time_driven_loop(&params, s_buf, r_buf);

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
