//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

#include <math.h>
#include <stdbool.h>
#include <assert.h>
#include <time.h>
#include <sys/time.h>
#include <inttypes.h>
#include <string.h>
#include <stdio.h>

#ifndef OPENHPCA_OVERLAP_H
#define OPENHPCA_OVERLAP_H

#define USE_CLOCK_GETTIME (0)
#define USE_MPI_WTIME (1)
#define USE_GETTIMEOFDAY (2)
#define TIMER (USE_CLOCK_GETTIME)

#define DEFAULT_WARMUP (100)
#define DEFAULT_MIN_ELTS (1)
#define DDM_DEFAULT_MAX_ELTS (131072)
#define TDM_DEFAULT_MAX_ELTS (1000000)
#define DDM_DEFAULT_N_ITERS (100)
#define TDM_DEFAULT_N_ITERS (25)
#define TDM_MAX_ITERS (50)
#define DEFAULT_NUM_VALIDATION_STEPS (2) // How many tests are necessary to deem an amount of work as not allowing perfect overlap
#define DEFAULT_CUTOFF_TIME (500)        // in milli-seconds
#define DEFAULT_OVERLAP_THRESHOLD (5)    // If the difference between the injected work that allows overlap and the one that does not allow overlap is x%, the result is precise enough and we stop
#define MAX_NUM_CALIBRATION_POINTS (1000)

#define OVERLAP_MIN_NUM_ELTS_ENVVAR "OPENHPCA_OVERLAP_MIN_NUM_ELTS"
#define OVERLAP_MAX_NUM_ELTS_ENVVAR "OPENHPCA_OVERLAP_MAX_NUM_ELTS"
#define OVERLAP_VALIDATION_STEPS_ENVVAR "OPENHPCA_OVERLAP_VALIDATION_STEPS"
#define OVERLAP_CALIBRATION_ENVVAR "OPENHPCA_OVERLAP_CALIBRATION"
#define OVERLAP_VERBOSE_ENVVAR "OPENHPCA_OVERLAP_VERBOSE"
#define OVERLAP_DEBUG_ENVVAR "OPENHPCA_OVERLAP_DEBUG"
#define OVERLAP_DATA_DRIVEN_MODEL_ENVVAR "OPENHPCA_DATA_DRIVEN_EXECUTION"
#define OVERLAP_CUTOFF_TIME_ENVVAR "OPENHPCA_OVERLAP_CUTOFF_TIME"
#define OVERLAP_DEFAULT_TDM_N_ITERS_ENVVAR "OPENHPCA_DEFAULT_TDM_NUM_ITERS"
#define OVERLAP_MAX_TDM_ITERS_ENVVAR "OPENHPCA_DEFAULT_TDM_NUM_ITERS"

#define OVERLAP_ACCEPTANCE_THRESHOLD_ENVVAR "OPENHPCA_OVERLAP_ACCEPTANCE_THRESHOLD"

#define asm __asm__

typedef struct overlap_params
{
    uint64_t min_elts;
    uint64_t max_elts;
    bool verbose;
    bool debug;
    int world_size;
    int world_rank;
    int calibration;
    int validation_steps;
    bool data_driven_model;
    int cutoff_time;
    int n_iters;
    int overlap_threshold;
    int max_iters;
} overlap_params_t;

typedef struct overlap_status
{
    // max_valid_work_units is the currently known maximum amount of work units that gives a valid overlap (i.e., does not increase the non-blocking collective overall time)
    int64_t max_valid_overlap_work_units;

    // min_invalid_overlap_work_units is the currently known minimum amount of work units that pushes the overall execution time with the work beyond the reference time
    int64_t min_invalid_overlap_work_units;

    int validation_units;
    int validation_count;
    int validation_threshold;
} overlap_status_t;

#ifdef __GNUC__
#define __no_optimization __attribute__((optimize("O0")))
#else
#define __no_optimization
#endif
void __no_optimization do_work(double x, double y, double a, double b, int64_t n_ops)
{
    if (n_ops == 0)
    {
        asm volatile("nop"); // No-op to avoid optimizations
        return;
    }

    for (x = 0; x < n_ops; x++)
    {
        asm volatile("nop"); // No-op to avoid optimizations
        y = a * (double)x + b;
    }
}

#define INIT_OVERLAP_LOOP                                                    \
    /* All the variables necessary to use non-blocking collectives */        \
    double end_time, total_time = 0.0, ref_time = 0.0, start_work, end_work; \
    double overlap;                                                          \
    int64_t work = 0;                                                        \
    int n_iters = DDM_DEFAULT_N_ITERS;                                       \
    MPI_Request req;                                                         \
    MPI_Status status;                                                       \
                                                                             \
    double *final_rank_times = NULL;                                         \
    double *final_work_stdevs = NULL;                                        \
    double *final_work_mins = NULL;                                          \
    double *final_work_maxs = NULL;                                          \
    double *final_work_totals = NULL;                                        \
    double *final_wait_stdevs = NULL;                                        \
    double *final_wait_mins = NULL;                                          \
    double *final_wait_maxs = NULL;                                          \
    double *final_wait_totals = NULL;                                        \
    double *final_post_stdevs = NULL;                                        \
    double *final_post_mins = NULL;                                          \
    double *final_post_maxs = NULL;                                          \
    double *final_post_totals = NULL;                                        \
                                                                             \
    overlap_status_t overlap_status;                                         \
                                                                             \
    double *ref_data, *data, *rank_times, *rank_ref_times, *rank_ref_stdevs; \
    MEMALLOC(ref_data, double, n_iters * sizeof(double));                    \
    MEMALLOC(data, double, n_iters * sizeof(double));                        \
    MEMALLOC(rank_times, double, params->world_size * sizeof(double));       \
    MEMALLOC(rank_ref_times, double, params->world_size * sizeof(double));   \
    MEMALLOC(rank_ref_stdevs, double, params->world_size * sizeof(double));  \
                                                                             \
    double *work_times, *work_stdevs, *work_mins, *work_maxs, *work_totals;  \
    MEMALLOC(work_times, double, n_iters * sizeof(double));                  \
    MEMALLOC(work_stdevs, double, params->world_size * sizeof(double));      \
    MEMALLOC(work_mins, double, params->world_size * sizeof(double));        \
    MEMALLOC(work_maxs, double, params->world_size * sizeof(double));        \
    MEMALLOC(work_totals, double, params->world_size * sizeof(double));      \
                                                                             \
    double *wait_times, *wait_stdevs, *wait_mins, *wait_maxs, *wait_totals;  \
    MEMALLOC(wait_times, double, n_iters * sizeof(double));                  \
    MEMALLOC(wait_stdevs, double, params->world_size * sizeof(double));      \
    MEMALLOC(wait_mins, double, params->world_size * sizeof(double));        \
    MEMALLOC(wait_maxs, double, params->world_size * sizeof(double));        \
    MEMALLOC(wait_totals, double, params->world_size * sizeof(double));      \
                                                                             \
    double *post_times, *post_stdevs, *post_mins, *post_maxs, *post_totals;  \
    MEMALLOC(post_times, double, n_iters * sizeof(double));                  \
    MEMALLOC(post_stdevs, double, params->world_size * sizeof(double));      \
    MEMALLOC(post_mins, double, params->world_size * sizeof(double));        \
    MEMALLOC(post_maxs, double, params->world_size * sizeof(double));        \
    MEMALLOC(post_totals, double, params->world_size * sizeof(double));      \
                                                                             \
    if (params->verbose && params->world_rank == 0)                          \
        display_overlap_params(params, sizeof(double));                      \
                                                                             \
    if (!calibrate(params))                                                  \
    {                                                                        \
        fprintf(stderr, "Calibration failed\n");                             \
        goto exit_error;                                                     \
    }                                                                        \
                                                                             \
    uint64_t n_elts;                                                         \
    int n;                                                                   \
    double stdev;

#define INIT_OVERLAP_BENCH           \
    overlap_params_t params;         \
                                     \
    int rc = MPI_Init(&argc, &argv); \
    if (MPI_SUCCESS != rc)           \
        goto exit_error;             \
    get_overlap_params(&params);

#define FINI_OVERLAP_BENCH          \
    do                              \
    {                               \
        MEMFREE(rank_times);        \
        MEMFREE(rank_ref_times);    \
        MEMFREE(rank_ref_stdevs);   \
                                    \
        MEMFREE(post_times);        \
        MEMFREE(post_stdevs);       \
        MEMFREE(post_mins);         \
        MEMFREE(post_maxs);         \
        MEMFREE(post_totals);       \
                                    \
        MEMFREE(work_times);        \
        MEMFREE(work_stdevs);       \
        MEMFREE(work_mins);         \
        MEMFREE(work_maxs);         \
        MEMFREE(work_totals);       \
                                    \
        MEMFREE(wait_times);        \
        MEMFREE(wait_stdevs);       \
        MEMFREE(wait_mins);         \
        MEMFREE(wait_maxs);         \
        MEMFREE(wait_totals);       \
                                    \
        MEMFREE(final_rank_times);  \
        MEMFREE(final_work_stdevs); \
        MEMFREE(final_work_mins);   \
        MEMFREE(final_work_maxs);   \
        MEMFREE(final_work_totals); \
        MEMFREE(final_wait_stdevs); \
        MEMFREE(final_wait_mins);   \
        MEMFREE(final_wait_maxs);   \
        MEMFREE(final_wait_totals); \
        MEMFREE(final_post_stdevs); \
        MEMFREE(final_post_mins);   \
        MEMFREE(final_post_maxs);   \
        MEMFREE(final_post_totals); \
                                    \
        MEMFREE(ref_data);          \
        MEMFREE(data);              \
    } while (0)

#define OVERLAP_DEBUG(_params, fmt, ...)                                      \
    do                                                                        \
    {                                                                         \
        if (_params->debug)                                                   \
            fprintf(stdout, "[%s:%d] " fmt, __FILE__, __LINE__, __VA_ARGS__); \
    } while (0)

#define MEMALLOC(_var, _type, _size)                                \
    do                                                              \
    {                                                               \
        _var = (_type *)malloc(_size);                              \
        if (_var == NULL)                                           \
        {                                                           \
            fprintf(stderr, "Out of resource (size=%ld)\n", _size); \
            goto exit_error;                                        \
        }                                                           \
    } while (0)

#define MEMFREE(_var)     \
    do                    \
    {                     \
        if (_var != NULL) \
        {                 \
            free(_var);   \
            _var = NULL;  \
        }                 \
    } while (0)

#define STDEV_UINT64(array, size, stdev)        \
    do                                          \
    {                                           \
        uint64_t _sum = 0;                      \
        double _mean;                           \
        stdev = 0.0;                            \
        int _i;                                 \
        for (_i = 1; _i < size; _i++)           \
            _sum += array[_i];                  \
        _mean = (double)_sum / size;            \
        for (_i = 0; _i < size; _i++)           \
        {                                       \
            stdev += pow(array[_i] - _mean, 2); \
        }                                       \
        stdev = sqrt(stdev / size);             \
    } while (0)

#define STDEV(array, size, stdev)               \
    do                                          \
    {                                           \
        double _sum = 0.0;                      \
        double _mean;                           \
        stdev = 0.0;                            \
        int _i;                                 \
        for (_i = 1; _i < size; _i++)           \
            _sum += array[_i];                  \
        _mean = _sum / size;                    \
        for (_i = 0; _i < size; _i++)           \
            stdev += pow(array[_i] - _mean, 2); \
        stdev = sqrt(stdev / size);             \
    } while (0)

// Overlap is defined as the ratio between the time spent
// to the work over the time spent to do the work in addition
// of the time for communication
#define GET_OVERLAP(_overlap, _total_time, _work_time) \
    do                                                 \
    {                                                  \
        _overlap = _work_time / _total_time;           \
    } while (0)

// All times are in milliseconds
#define GET_WORK_EQUIVALENCE(x, y, a, b, time, work)            \
    do                                                          \
    {                                                           \
        int64_t _w = 1;                                         \
        double _t = 0.0;                                        \
        while (_t < time)                                       \
        {                                                       \
            double _s = MPI_Wtime();                            \
            do_work(x, y, a, b, _w);                            \
            double _e = MPI_Wtime();                            \
            _t = _e - _s;                                       \
            _t *= 1000;                                         \
            if (_t < time)                                      \
            {                                                   \
                if (time / _t > 10)                             \
                    _w *= 2;                                    \
                else                                            \
                    _w += _w / 2; /* Add another 50% of work */ \
            }                                                   \
        }                                                       \
        work = _w;                                              \
    } while (0)

#define COMPUTE_REQUIRED_WORK                                                                                 \
    do                                                                                                        \
    {                                                                                                         \
        ref_time = total_time; /* at this point, ref_time is the local reference time */                      \
        STDEV(ref_data, n_iters, stdev);                                                                      \
        /* Gather reference data from all ranks so we can have a more accurate reference number */            \
        MPI_CHECK(MPI_Gather(&ref_time, 1, MPI_DOUBLE, rank_ref_times, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD));    \
        MPI_CHECK(MPI_Gather(&stdev, 1, MPI_DOUBLE, rank_ref_stdevs, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD));      \
                                                                                                              \
        if (params->world_rank == 0)                                                                          \
        {                                                                                                     \
            /* Calculate the global reference time */                                                         \
            ref_time = 0.0;                                                                                   \
            for (n = 0; n < params->world_size; n++)                                                          \
                ref_time += rank_ref_times[n];                                                                \
            ref_time /= params->world_size;                                                                   \
            /* Once we have the global reference time, we can estimate the work equivalence */                \
            GET_WORK_EQUIVALENCE(x, y, a, b, ref_time, work);                                                 \
            OVERLAP_DEBUG(params, "Work equivalence: %f seconds - %" PRId64 " work units\n", ref_time, work); \
        }                                                                                                     \
                                                                                                              \
        MPI_CHECK(MPI_Bcast(&work, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD));                                        \
        MPI_CHECK(MPI_Barrier(MPI_COMM_WORLD));                                                               \
    } while (0)

#define PROCESS_DATA                                                                                               \
    do                                                                                                             \
    {                                                                                                              \
        if (params->world_rank == 0)                                                                               \
        {                                                                                                          \
            if (total_time > ref_time + stdev && work > 1)                                                         \
            {                                                                                                      \
                /* Too much work */                                                                                \
                work = updated_overlap_status(params, &overlap_status, total_time, ref_time + stdev, false, work); \
                OVERLAP_DEBUG(params, "Too much work (%f >= %f), next trying with %" PRId64 " units\n",            \
                              total_time, ref_time + stdev, work);                                                 \
            }                                                                                                      \
                                                                                                                   \
            if (work == 1 && total_time > ref_time)                                                                \
            {                                                                                                      \
                OVERLAP_DEBUG(params, "No overlap (%f > %f)", total_time, ref_time);                               \
                PRINT_STATS;                                                                                       \
                work = -1;                                                                                         \
            }                                                                                                      \
        }                                                                                                          \
    } while (0)

#define MPI_CHECK(exp)                                                             \
    do                                                                             \
    {                                                                              \
        if (exp != MPI_SUCCESS)                                                    \
        {                                                                          \
            fprintf(stderr, "[%s:%d] MPI operation failed\n", __func__, __LINE__); \
            goto exit_error;                                                       \
        }                                                                          \
    } while (0)

#define PRINT_STATS                                                                                                                                                           \
    do                                                                                                                                                                        \
    {                                                                                                                                                                         \
        if (params->verbose)                                                                                                                                                  \
        {                                                                                                                                                                     \
            fprintf(stdout, "Total execution times (%d iterations) <(data size)/(work units injected)/(reference iteration time)/stdev [rank execution times]>:\n", n_iters); \
            fprintf(stdout, "%ld/%" PRId64 "/%f/%f ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units, ref_time, stdev);                                 \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_rank_times[n]);                                                                                                                  \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nTotal post time (%d iterations) <(data size)/(work units injected) [rank post times]>:\n", n_iters);                                           \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_post_totals[n]);                                                                                                                 \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nTotal work time (%d iterations) <(data size)/(work units injected) [rank work time]>:\n", n_iters);                                            \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_work_totals[n]);                                                                                                                 \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nTotal wait time (%d iterations) <(data size)/(work units injected) [rank wait times]>:\n", n_iters);                                           \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_wait_totals[n]);                                                                                                                 \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nPost stdev <(data size)/(work units injected) [rank post stdevs]>:\n");                                                                        \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_post_stdevs[n]);                                                                                                                 \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nPost mins per iteration <(data size)/(work units injected) [rank post mins]>:\n");                                                             \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_post_mins[n]);                                                                                                                   \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nPost maxs per iteration <(data size)/(work units injected) [rank post maxs]>:\n");                                                             \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_post_maxs[n]);                                                                                                                   \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nWork stdev <(data size)/(work units injected) [rank work stdevs]>:\n");                                                                        \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_work_stdevs[n]);                                                                                                                 \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nWork mins per iteration <(data size)/(work units injected) [rank work mins]>:\n");                                                             \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_work_mins[n]);                                                                                                                   \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nWork maxs per iteration <(data size)/(work units injected) [rank work maxs]>:\n");                                                             \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_work_maxs[n]);                                                                                                                   \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nWait stdev <(data size)/(work units injected) [rank wait stdevs]>:\n");                                                                        \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_wait_stdevs[n]);                                                                                                                 \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nWait mins per iteration <(data size)/(work units injected) [rank wait mins]>:\n");                                                             \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_wait_mins[n]);                                                                                                                   \
            fprintf(stdout, "\n");                                                                                                                                            \
                                                                                                                                                                              \
            fprintf(stdout, "\nWait maxs per iteration <(data size)/(work units injected) [rank wait maxs]>:\n");                                                             \
            fprintf(stdout, "%ld/%" PRId64 " ", n_elts * sizeof(double), overlap_status.max_valid_overlap_work_units);                                                        \
            for (n = 0; n < params->world_size; n++)                                                                                                                          \
                fprintf(stdout, "%f ", final_wait_maxs[n]);                                                                                                                   \
            fprintf(stdout, "\n");                                                                                                                                            \
        }                                                                                                                                                                     \
    } while (0)

static void get_overlap_params(overlap_params_t *params)
{
    char *min_elts_str = getenv(OVERLAP_MIN_NUM_ELTS_ENVVAR);
    char *max_elts_str = getenv(OVERLAP_MAX_NUM_ELTS_ENVVAR);
    char *calibration_str = getenv(OVERLAP_CALIBRATION_ENVVAR);
    char *verbose_str = getenv(OVERLAP_VERBOSE_ENVVAR);
    char *debug_str = getenv(OVERLAP_DEBUG_ENVVAR);
    char *validation_steps_str = getenv(OVERLAP_VALIDATION_STEPS_ENVVAR);
    char *data_driven_model_str = getenv(OVERLAP_DATA_DRIVEN_MODEL_ENVVAR);
    char *cutoff_time_str = getenv(OVERLAP_CUTOFF_TIME_ENVVAR);
    char *default_n_iters_str = getenv(OVERLAP_DEFAULT_TDM_N_ITERS_ENVVAR);
    char *overlap_threshold_str = getenv(OVERLAP_ACCEPTANCE_THRESHOLD_ENVVAR);
    char *max_iters_str = getenv(OVERLAP_MAX_TDM_ITERS_ENVVAR);

    /* Initialize to default values */
    params->verbose = 0;
    params->debug = 0;
    params->calibration = 0;
    params->validation_steps = DEFAULT_NUM_VALIDATION_STEPS;
    params->data_driven_model = false;
    params->cutoff_time = DEFAULT_CUTOFF_TIME;
    params->min_elts = DEFAULT_MIN_ELTS;
    params->overlap_threshold = DEFAULT_OVERLAP_THRESHOLD;
    params->max_iters = TDM_MAX_ITERS;
    if (params->data_driven_model)
    {
        params->max_elts = DDM_DEFAULT_MAX_ELTS;
        params->n_iters = DDM_DEFAULT_N_ITERS;
    }
    else
    {
        params->max_elts = TDM_DEFAULT_MAX_ELTS;
        params->n_iters = TDM_DEFAULT_N_ITERS;
    }

    int size, rank;
    MPI_Comm_size(MPI_COMM_WORLD, &size);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    params->world_size = size;
    params->world_rank = rank;

    if (min_elts_str)
        params->min_elts = atoi(min_elts_str);

    if (max_elts_str)
        params->max_elts = atoi(max_elts_str);

    if (verbose_str)
    {
        int v = atoi(verbose_str);
        if (v)
            params->verbose = true;
    }

    if (debug_str)
    {
        int v = atoi(debug_str);
        if (v)
            params->debug = true;
    }

    if (calibration_str)
        params->calibration = atoi(calibration_str);

    if (validation_steps_str)
        params->validation_steps = atoi(validation_steps_str);

    if (data_driven_model_str)
    {
        int v = atoi(data_driven_model_str);
        if (v)
            params->data_driven_model = true;
    }

    if (cutoff_time_str)
    {
        int v = atoi(cutoff_time_str);
        if (v > 0)
            params->cutoff_time = v;
    }

    if (default_n_iters_str)
    {
        int v = atoi(default_n_iters_str);
        if (v > 0)
            params->n_iters = v;
    }

    if (overlap_threshold_str)
    {
        int v = atoi(overlap_threshold_str);
        if (v > 0)
            params->overlap_threshold = v;
    }

    if (max_iters_str)
    {
        int v = atoi(max_iters_str);
        if (v > 0)
            params->max_iters = v;
    }
}

#define MINMAX(array, sz, min, max) \
    do                              \
    {                               \
        min = -1.0;                 \
        max = -1.0;                 \
        int _i;                     \
        for (_i = 0; _i < sz; _i++) \
        {                           \
            if (min == -1.0)        \
            {                       \
                min = array[_i];    \
                max = array[_i];    \
            }                       \
            if (min > array[_i])    \
                min = array[_i];    \
            if (max < array[_i])    \
                max = array[_i];    \
        }                           \
    } while (0)

static void display_overlap_params(overlap_params_t *params, size_t dtsz)
{
    if (params == NULL)
        return;

    fprintf(stdout, "Parameters:\n");
    fprintf(stdout, "Minimum number of elements exchanged: %" PRIu64 " (%ld bytes)\n", params->min_elts, params->min_elts * dtsz);
    fprintf(stdout, "Maximim number of elements exchanged: %" PRIu64 " (%ld bytes)\n", params->max_elts, params->max_elts * dtsz);
    fprintf(stdout, "Validation steps: %d\n", params->validation_steps);
    if (params->calibration)
        fprintf(stdout, "Calibration: ON\n");
    else
        fprintf(stdout, "Calibration: OFF\n");
    if (params->debug)
        fprintf(stdout, "Debug mode: ON\n");
    else
        fprintf(stdout, "Debug mode: OFF\n");
    if (params->data_driven_model)
        fprintf(stdout, "Data driven execution: ON\n");
    else
        fprintf(stdout, "Time driven execution: ON\n");
    fprintf(stdout, "\n");
}

#if TIMER == USE_CLOCK_GETTIME
#define TIMESTAMP(time)                                              \
    do                                                               \
    {                                                                \
        struct timespec timer;                                       \
        clock_gettime(CLOCK_PROCESS_CPUTIME_ID, &timer);             \
        time = (uint64_t)(timer.tv_sec * 1e9 + timer.tv_nsec) / 1e3; \
    } while (0)
#elif TIMER == USE_GETTIMEOFDAY
#define TIMESTAMP(time)                                  \
    do                                                   \
    {                                                    \
        struct timeval tv;                               \
        gettimeofday(&tv, NULL);                         \
        return (uint64_t)(tv.tv_sec * 1e6 + tv.tv_usec); \
    } while (0)
#elif TIMER == USE_MPI_WTIME
#define TIMESTAMP(time)                       \
    do                                        \
    {                                         \
        time = (uint64_t)(MPI_Wtime() * 1e6); \
    } while (0)
#else
#define TIMESTAMP(time) (fprintf(stderr, "No timer method defined"))
#endif

static bool calibrate_latencies(overlap_params_t *params)
{
    bool ret = true;
    size_t count = 0;
    MPI_Request req;
    MPI_Status status;
    double start_time, end_time;

    double *val = (double *)malloc(params->max_elts * sizeof(double));
    if (val == NULL)
    {
        ret = false;
        goto exit_error;
    }
    double *result = (double *)malloc(params->max_elts * sizeof(double));
    if (result == NULL)
    {
        ret = false;
        goto exit_error;
    }

    int i, j;
    double timing;
    if (params->world_rank == 0)
        fprintf(stdout, "Message size\tlatency (us)\n");
    for (i = params->min_elts; i <= params->max_elts; i *= 2)
    {
        MPI_Barrier(MPI_COMM_WORLD);
        // warmup
        for (j = 0; j < 100; j++)
        {
            MPI_CHECK(MPI_Iallreduce(val, result, i, MPI_DOUBLE, MPI_SUM, MPI_COMM_WORLD, &req));
            MPI_CHECK(MPI_Wait(&req, &status));
            MPI_Barrier(MPI_COMM_WORLD);
        }

        timing = 0.0;
        count = 0;
        for (j = 0; j < 1000; j++)
        {
            start_time = MPI_Wtime();
            MPI_CHECK(MPI_Iallreduce(val, result, i, MPI_DOUBLE, MPI_SUM, MPI_COMM_WORLD, &req));
            MPI_CHECK(MPI_Wait(&req, &status));
            end_time = MPI_Wtime();
            timing += end_time - start_time;
            count++;
            MPI_Barrier(MPI_COMM_WORLD);
        }
        double latency = (timing * 1e6) / (double)count;
        double total_latency;
        MPI_CHECK(MPI_Reduce(&latency, &total_latency, 1, MPI_DOUBLE, MPI_SUM, 0, MPI_COMM_WORLD));
        if (params->world_rank == 0)
            fprintf(stdout, "%ld\t\t%f\n", i * sizeof(double), total_latency / params->world_size);
        MPI_Barrier(MPI_COMM_WORLD);
    }

exit_error:
    if (val)
        free(val);
    if (result)
        free(result);
    return ret;
}

static bool calibrate_mpi_wait(overlap_params_t *params)
{
    bool ret = true;
    uint64_t test_count = 0;
    MPI_Request req;
    MPI_Status status;

    double *val = (double *)malloc(params->max_elts * sizeof(double));
    if (val == NULL)
    {
        ret = false;
        goto exit_error;
    }
    double *result = (double *)malloc(params->max_elts * sizeof(double));
    if (result == NULL)
    {
        ret = false;
        goto exit_error;
    }

    int i, j;
    int completed;
    uint64_t max_time, last_time, total_time;
    uint64_t start, end, timer;
    uint64_t *total_times = (uint64_t *)malloc(params->world_size * sizeof(uint64_t));
    if (total_times == NULL)
        goto exit_error;
    uint64_t *max_times = (uint64_t *)malloc(params->world_size * sizeof(uint64_t));
    if (max_times == NULL)
        goto exit_error;
    uint64_t *last_times = (uint64_t *)malloc(params->world_size * sizeof(uint64_t));
    if (last_times == NULL)
        goto exit_error;
    uint64_t *test_counts = (uint64_t *)malloc(params->world_size * sizeof(uint64_t));
    if (test_counts == NULL)
        goto exit_error;

    for (i = params->min_elts; i <= params->max_elts; i *= 2)
    {
        MPI_Barrier(MPI_COMM_WORLD);
        // warmup
        for (j = 0; j < 100; j++)
        {
            MPI_CHECK(MPI_Iallreduce(val, result, i, MPI_DOUBLE, MPI_SUM, MPI_COMM_WORLD, &req));
            MPI_CHECK(MPI_Wait(&req, &status));
            MPI_Barrier(MPI_COMM_WORLD);
        }

        {
            max_time = 0;
            test_count = 0;
            total_time = 0;
            MPI_CHECK(MPI_Iallreduce(val, result, i, MPI_DOUBLE, MPI_SUM, MPI_COMM_WORLD, &req));
            do
            {
                TIMESTAMP(start);
                MPI_CHECK(MPI_Test(&req, &completed, &status));
                TIMESTAMP(end);
                timer = end - start;
                if (max_time < timer)
                    max_time = timer;
                if (completed)
                    last_time = timer;
                total_time += timer;
                test_count++;
            } while (!completed);
            MPI_Barrier(MPI_COMM_WORLD);
        }
        MPI_Barrier(MPI_COMM_WORLD);
        MPI_Gather(&total_time, 1, MPI_UINT64_T, total_times, 1, MPI_UINT64_T, 0, MPI_COMM_WORLD);
        MPI_Gather(&max_time, 1, MPI_UINT64_T, max_times, 1, MPI_UINT64_T, 0, MPI_COMM_WORLD);
        MPI_Gather(&last_time, 1, MPI_UINT64_T, last_times, 1, MPI_UINT64_T, 0, MPI_COMM_WORLD);
        MPI_Gather(&test_count, 1, MPI_UINT64_T, test_counts, 1, MPI_UINT64_T, 0, MPI_COMM_WORLD);

        if (params->world_rank == 0)
        {
            fprintf(stdout, "Total test times per rank (us) - 1 iteration\n");
            for (j = 0; j < params->world_size; j++)
                fprintf(stdout, "%" PRIu64 " ", total_times[j]);
            fprintf(stdout, "\n");

            fprintf(stdout, "Max test times per rank (us) - 1 iteration\n");
            for (j = 0; j < params->world_size; j++)
                fprintf(stdout, "%" PRIu64 " ", max_times[j]);
            fprintf(stdout, "\n");

            fprintf(stdout, "Last test times per rank (us) - 1 iteration\n");
            for (j = 0; j < params->world_size; j++)
                fprintf(stdout, "%" PRIu64 " ", total_times[j]);
            fprintf(stdout, "\n");

            fprintf(stdout, "Test counts per rank (us) - 1 iteration\n");
            for (j = 0; j < params->world_size; j++)
                fprintf(stdout, "%" PRIu64 " ", test_counts[j]);
            fprintf(stdout, "\n");
        }
    }

    if (val)
        free(val);
    if (result)
        free(result);
    if (total_times)
        free(total_times);
    if (max_times)
        free(max_times);
    if (last_times)
        free(last_times);
    if (test_counts)
        free(test_counts);
    ret = false; // debug: should be true
    return ret;

exit_error:
    if (val)
        free(val);
    if (result)
        free(result);
    if (total_times)
        free(total_times);
    if (max_times)
        free(max_times);
    if (last_times)
        free(last_times);
    if (test_counts)
        free(test_counts);
    return ret;
}

static bool calibrate_collectives(overlap_params_t *params)
{
    if (!calibrate_mpi_wait(params))
        return false;

    if (!calibrate_latencies(params))
        return false;

    return true;
}

// sync_params lets us make sure that regardless of the MPI implementation
// that is used, all ranks have a consistent set of parameters, regardless
// of how environment variables are handled
static inline bool sync_params(overlap_params_t *params)
{
    if (params == NULL)
        return false;

    MPI_CHECK(MPI_Bcast(&(params->verbose), 1, MPI_C_BOOL, 0, MPI_COMM_WORLD));
    MPI_CHECK(MPI_Bcast(&(params->debug), 1, MPI_C_BOOL, 0, MPI_COMM_WORLD));
    MPI_CHECK(MPI_Bcast(&(params->calibration), 1, MPI_C_BOOL, 0, MPI_COMM_WORLD));
    MPI_CHECK(MPI_Bcast(&(params->min_elts), 1, MPI_INT, 0, MPI_COMM_WORLD));
    MPI_CHECK(MPI_Bcast(&(params->max_elts), 1, MPI_INT, 0, MPI_COMM_WORLD));
    MPI_CHECK(MPI_Bcast(&(params->validation_steps), 1, MPI_INT, 0, MPI_COMM_WORLD));
    return true;
exit_error:
    return false;
}

static bool calibrate(overlap_params_t *params)
{
    if (!sync_params(params))
        return false;

    if (!params->calibration)
        return true;

    if (!calibrate_collectives(params))
        return false;

    return true;
}

#define INIT_OVERLAP_STATUS(params, status)                      \
    do                                                           \
    {                                                            \
        status->max_valid_overlap_work_units = -1;               \
        status->min_invalid_overlap_work_units = -1;             \
        status->validation_units = -1;                           \
        status->validation_count = 0;                            \
        status->validation_threshold = params->validation_steps; \
    } while (0)

static int updated_overlap_status(overlap_params_t *params, overlap_status_t *status, double run_time, double ref_time, bool passed, int work_units)
{
    OVERLAP_DEBUG(params, "Updating with %d work units (passed=%d)\n", work_units, passed);

    if (!passed)
    {
        if (status->validation_units == -1)
            status->validation_units = work_units;
        status->validation_count++;
        if (status->validation_count >= status->validation_threshold)
        {
            // We had enough runs with the same result, it is validated
            status->validation_units = -1;
            status->validation_count = 0;
        }
        else
        {
            double ratio = run_time / ref_time;
            if (ratio > 10)
            {
                OVERLAP_DEBUG(params, "Execution time high above target, trying with %d work units\n", work_units / 10);
                status->validation_count = 0;
                return work_units / 10;
            }

            if (ratio > 2)
            {
                OVERLAP_DEBUG(params, "Execution time high above target, trying with %d work units\n", work_units / 2);
                status->validation_count = 0;
                return work_units / 2;
            }

            OVERLAP_DEBUG(params, "Running the same configuration again for validation (%d work units)\n", work_units);
            return work_units;
        }
    }

    if (passed && status->validation_units != -1)
    {
        OVERLAP_DEBUG(params, "Mmmm it seems we have some jitter, we have one result (step %d), resuming...\n", status->validation_count);
        status->validation_units = -1;
        status->validation_count = 0;
        if (status->max_valid_overlap_work_units == status->min_invalid_overlap_work_units)
        {
            // This is a special case: a configuration that was not validated before now has a positive result
            // and there are in a configuration where we cannot test anything else. So we slightly increase the
            // upper limit so we can continue to find the best configuration.
            status->min_invalid_overlap_work_units = status->max_valid_overlap_work_units + 2;
        }
    }

    switch (passed)
    {
    case true:
        // the overlap test succeeded (time was bellow reference time)
        if (status->max_valid_overlap_work_units < work_units)
            status->max_valid_overlap_work_units = work_units;
        break;
    default:
        // the overlap test failed (time was above reference time)
        if (status->min_invalid_overlap_work_units == -1 || status->min_invalid_overlap_work_units > work_units)
            status->min_invalid_overlap_work_units = work_units;
    }

    OVERLAP_DEBUG(params, "Max=%" PRId64 "; Min=%" PRId64 "\n", status->min_invalid_overlap_work_units, status->max_valid_overlap_work_units);

    // figure out the next amount of work units to test
    if (status->max_valid_overlap_work_units == -1 && status->min_invalid_overlap_work_units == -1)
    {
        // error
        return -1;
    }

    if (status->min_invalid_overlap_work_units == -1)
    {
        // We did not figure out yet the higher bound
        OVERLAP_DEBUG(params, "We did not figure out yet the higher bound (current work units=%d)\n", work_units);
        return status->max_valid_overlap_work_units * 2;
    }

    if (status->max_valid_overlap_work_units == -1)
    {
        // We did not figure out yet the lower bound
        OVERLAP_DEBUG(params, "We did not figure out yet the lower bound (current work units=%d)\n", work_units);
        /* Compute by how much we need to decrease the time */
        double ratio = run_time / ref_time;
        if (ratio > 10)
            return status->min_invalid_overlap_work_units / 10;

        return status->min_invalid_overlap_work_units / 2;
    }

    if (status->min_invalid_overlap_work_units != -1 && status->max_valid_overlap_work_units != -1)
    {
        if (status->min_invalid_overlap_work_units - status->max_valid_overlap_work_units <= 0)
            return -1; // error, it should not happen
        if (status->min_invalid_overlap_work_units - status->max_valid_overlap_work_units < 1)
            return 0; // no more work to do
        return (status->max_valid_overlap_work_units + (status->min_invalid_overlap_work_units - status->max_valid_overlap_work_units) / 2);
    }

    return 0;
}

static bool check_results(overlap_params_t *params, double ref_time, double *ref_times, double *rank_ref_stdevs, double *rank_times, double *work_times, double *work_stdevs)
{
    int i;
    double min_stdev, max_stdev;
    MINMAX(rank_ref_stdevs, params->world_size, min_stdev, max_stdev);
    double mean_work_time = 0.0;
    for (i = 0; i < params->world_size; i++)
        mean_work_time += work_times[i];
    mean_work_time /= params->world_size;

    // The average work time cannot be greater than the reference time plus the standard deviation
    if (work_times != NULL && ref_times != NULL && rank_ref_stdevs != NULL && ref_times != NULL && rank_ref_stdevs != NULL && mean_work_time > ref_time + max_stdev)
    {
        fprintf(stderr, "Rank %d: %f is greater than %f + %f = %f\n", i, work_times[i], ref_times[i], rank_ref_stdevs[i], ref_times[i] + rank_ref_stdevs[i]);
        return false;
    }

    return true;
}

#endif
