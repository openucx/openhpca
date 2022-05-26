//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

#ifndef OVERLAP_DDM_H_
#define OVERLAP_DDM_H_

#define DDM_VARIABLES                                  \
    double start_time, end_post, start_wait;           \
    double work_stdev, work_min, work_max, work_total; \
    double wait_stdev, wait_min, wait_max, wait_total; \
    double post_stdev, post_min, post_max, post_total; \
    int warmup = DEFAULT_WARMUP;

static inline bool
ddm_compute_overlap(overlap_params_t *params, overlap_status_t *status, double ref_time, double *ref_times, double *rank_ref_stdevs, double *rank_times, double *work_times, double *work_stdevs, double *overlap)
{
    if (status->max_valid_overlap_work_units == -1)
    {
        // We could not find any overlap
        *overlap = 0.0;
        return true;
    }

    if (params->data_driven_model && !check_results(params, ref_time, ref_times, rank_ref_stdevs, rank_times, work_times, work_stdevs))
    {
        fprintf(stderr, "Invalid results\n");
        return false;
    }

    // We now know that the data is valid. It for instance means that even if some work times have been reported as higher
    // than the reference time, it is all within statistical acceptance.

    // Calculate the mean work time across all the ranks
    int i;
    double mean_work_time = 0.0;
    for (i = 0; i < params->world_size; i++)
        mean_work_time += work_times[i];
    mean_work_time /= params->world_size;

    // Calculate the mean reference time across all the ranks
    double mean_ref_time = 0.0;
    for (i = 0; i < params->world_size; i++)
        mean_ref_time += ref_times[i];
    mean_ref_time /= params->world_size;

    if (mean_work_time >= mean_ref_time)
        *overlap = 100;
    else
        *overlap = mean_work_time / mean_ref_time * 100;

    return true;
}

static inline double
ddm_data_process(overlap_params_t *params, double *rank_times)
{
    double total_time = 0.0;
    if (params->world_rank == 0)
    {
        /* total_time is so far a local value, we calculate the mean of the value from all ranks */
        double mean = 0.0;
        int n;
        for (n = 0; n < params->world_size; n++)
            mean += rank_times[n];
        mean /= params->world_size;
        total_time = mean;
    }
    return total_time;
}

#define DDM_GATHER_AND_PROCESS_DATA                                                                                                                                         \
    do                                                                                                                                                                      \
    {                                                                                                                                                                       \
        STDEV(work_times, n_iters, work_stdev);                                                                                                                             \
        MINMAX(work_times, n_iters, work_min, work_max);                                                                                                                    \
                                                                                                                                                                            \
        STDEV(wait_times, n_iters, wait_stdev);                                                                                                                             \
        MINMAX(wait_times, n_iters, wait_min, wait_max);                                                                                                                    \
                                                                                                                                                                            \
        STDEV(post_times, n_iters, post_stdev);                                                                                                                             \
        MINMAX(post_times, n_iters, post_min, post_max);                                                                                                                    \
                                                                                                                                                                            \
        MPI_Barrier(MPI_COMM_WORLD);                                                                                                                                        \
        MPI_Gather(&total_time, 1, MPI_DOUBLE, rank_times, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                               \
        MPI_Gather(&work_stdev, 1, MPI_DOUBLE, work_stdevs, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                              \
        MPI_Gather(&work_min, 1, MPI_DOUBLE, work_mins, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                                  \
        MPI_Gather(&work_max, 1, MPI_DOUBLE, work_maxs, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                                  \
        MPI_Gather(&work_total, 1, MPI_DOUBLE, work_totals, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                              \
        MPI_Gather(&wait_stdev, 1, MPI_DOUBLE, wait_stdevs, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                              \
        MPI_Gather(&wait_min, 1, MPI_DOUBLE, wait_mins, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                                  \
        MPI_Gather(&wait_max, 1, MPI_DOUBLE, wait_maxs, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                                  \
        MPI_Gather(&wait_total, 1, MPI_DOUBLE, wait_totals, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                              \
        MPI_Gather(&post_stdev, 1, MPI_DOUBLE, post_stdevs, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                              \
        MPI_Gather(&post_min, 1, MPI_DOUBLE, post_mins, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                                  \
        MPI_Gather(&post_max, 1, MPI_DOUBLE, post_maxs, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                                  \
        MPI_Gather(&post_total, 1, MPI_DOUBLE, post_totals, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD);                                                                              \
                                                                                                                                                                            \
        total_time = ddm_data_process(params, rank_times);                                                                                                                  \
        PROCESS_DATA;                                                                                                                                                       \
        if (params->world_rank == 0 && total_time <= ref_time + stdev)                                                                                                      \
        {                                                                                                                                                                   \
            /* Overlap okay, refining results */                                                                                                                            \
            work = updated_overlap_status(params, &overlap_status, total_time, ref_time + stdev, true, work);                                                               \
            OVERLAP_DEBUG(params, "Overlap okay, refining results with %" PRId64 " units\n", work);                                                                         \
            /* Saving data since it could be the final results */                                                                                                           \
            if (final_rank_times == NULL)                                                                                                                                   \
                MEMALLOC(final_rank_times, double, params->world_size * sizeof(double));                                                                                    \
            if (final_work_stdevs == NULL)                                                                                                                                  \
                MEMALLOC(final_work_stdevs, double, params->world_size * sizeof(double));                                                                                   \
            if (final_work_mins == NULL)                                                                                                                                    \
                MEMALLOC(final_work_mins, double, params->world_size * sizeof(double));                                                                                     \
            if (final_work_maxs == NULL)                                                                                                                                    \
                MEMALLOC(final_work_maxs, double, params->world_size * sizeof(double));                                                                                     \
            if (final_work_totals == NULL)                                                                                                                                  \
                MEMALLOC(final_work_totals, double, params->world_size * sizeof(double));                                                                                   \
            if (final_wait_stdevs == NULL)                                                                                                                                  \
                MEMALLOC(final_wait_stdevs, double, params->world_size * sizeof(double));                                                                                   \
            if (final_wait_mins == NULL)                                                                                                                                    \
                MEMALLOC(final_wait_mins, double, params->world_size * sizeof(double));                                                                                     \
            if (final_wait_maxs == NULL)                                                                                                                                    \
                MEMALLOC(final_wait_maxs, double, params->world_size * sizeof(double));                                                                                     \
            if (final_wait_totals == NULL)                                                                                                                                  \
                MEMALLOC(final_wait_totals, double, params->world_size * sizeof(double));                                                                                   \
            if (final_post_stdevs == NULL)                                                                                                                                  \
                MEMALLOC(final_post_stdevs, double, params->world_size * sizeof(double));                                                                                   \
            if (final_post_mins == NULL)                                                                                                                                    \
                MEMALLOC(final_post_mins, double, params->world_size * sizeof(double));                                                                                     \
            if (final_post_maxs == NULL)                                                                                                                                    \
                MEMALLOC(final_post_maxs, double, params->world_size * sizeof(double));                                                                                     \
            if (final_post_totals == NULL)                                                                                                                                  \
                MEMALLOC(final_post_totals, double, params->world_size * sizeof(double));                                                                                   \
            memcpy(final_rank_times, rank_times, params->world_size * sizeof(double));                                                                                      \
            memcpy(final_work_stdevs, work_stdevs, params->world_size * sizeof(double));                                                                                    \
            memcpy(final_work_mins, work_mins, params->world_size * sizeof(double));                                                                                        \
            memcpy(final_work_maxs, work_maxs, params->world_size * sizeof(double));                                                                                        \
            memcpy(final_work_totals, work_totals, params->world_size * sizeof(double));                                                                                    \
            memcpy(final_wait_stdevs, wait_stdevs, params->world_size * sizeof(double));                                                                                    \
            memcpy(final_wait_mins, wait_mins, params->world_size * sizeof(double));                                                                                        \
            memcpy(final_wait_maxs, wait_maxs, params->world_size * sizeof(double));                                                                                        \
            memcpy(final_wait_totals, wait_totals, params->world_size * sizeof(double));                                                                                    \
            memcpy(final_post_stdevs, post_stdevs, params->world_size * sizeof(double));                                                                                    \
            memcpy(final_post_mins, post_mins, params->world_size * sizeof(double));                                                                                        \
            memcpy(final_post_maxs, post_maxs, params->world_size * sizeof(double));                                                                                        \
            memcpy(final_post_totals, post_totals, params->world_size * sizeof(double));                                                                                    \
        }                                                                                                                                                                   \
                                                                                                                                                                            \
        if (work == -1)                                                                                                                                                     \
        {                                                                                                                                                                   \
            PRINT_STATS;                                                                                                                                                    \
            if (!ddm_compute_overlap(params, &overlap_status, ref_time, rank_ref_times, rank_ref_stdevs, final_rank_times, final_work_totals, final_work_stdevs, &overlap)) \
                goto exit_error;                                                                                                                                            \
            fprintf(stdout, "%ld\t%f\n", n_elts * sizeof(double), overlap);                                                                                                 \
        }                                                                                                                                                                   \
    } while (0)

#endif // OVERLAP_DDM_H_