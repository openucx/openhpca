//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

#ifndef OVERLAP_TDM_H_
#define OVERLAP_TDM_H_

#define TDM_VARIABLES \
    int done = 0;

#define TDM_SET_ITERS_AND_ELTS                                                                                                                     \
    if (params->world_rank == 0)                                                                                                                   \
        OVERLAP_DEBUG(params, "Concensus is n_iter = %d; n_elts = %" PRIu64 " with time = %f and stdev = %f\n", n_iters, n_elts, ref_time, stdev); \
                                                                                                                                                   \
    if (n_elts >= params->max_elts)                                                                                                                \
    {                                                                                                                                              \
        OVERLAP_DEBUG(params, "Scale is too small, unable to compute overlap (n_elts=%" PRIu64 ")\n", n_elts);                                     \
        goto exit_error;                                                                                                                           \
    }                                                                                                                                              \
                                                                                                                                                   \
    if (params->world_rank == 0)                                                                                                                   \
    {                                                                                                                                              \
        OVERLAP_DEBUG(params, "Getting work equivalence for time of %f\n", ref_time);                                                              \
        GET_WORK_EQUIVALENCE(x, y, a, b, ref_time, work);                                                                                          \
        OVERLAP_DEBUG(params, "Work equivalent is %" PRId64 " units of work (time = %f)\n", work, ref_time);                                       \
    }                                                                                                                                              \
    MPI_CHECK(MPI_Bcast(&work, 1, MPI_DOUBLE, 0, MPI_COMM_WORLD));                                                                                 \
    MPI_CHECK(MPI_Barrier(MPI_COMM_WORLD));                                                                                                        \
    ref_work = work;

#define TDM_PROCESS_DATA                                                                                                                              \
    if (params->world_rank == 0)                                                                                                                      \
    {                                                                                                                                                 \
        total_time /= n_iters;                                                                                                                        \
        OVERLAP_DEBUG(params, "Processing data: ref_time = %f; current time = %f, stdev = %f\n", ref_time, total_time, stdev);                        \
        PROCESS_DATA;                                                                                                                                 \
                                                                                                                                                      \
        if (total_time <= ref_time + stdev)                                                                                                           \
        {                                                                                                                                             \
            if (work >= ref_work)                                                                                                                     \
            {                                                                                                                                         \
                /* Overlap okay and we have at least the same amount of work so we are done */                                                        \
                OVERLAP_DEBUG(params, "Overlap okay with work = %" PRId64 " and ref work = %" PRId64 "\n", work, ref_work);                           \
                overlap = 100;                                                                                                                        \
                final_work_time = work_time;                                                                                                          \
                work = -1; /* This means we are done and will stop all the ranks */                                                                   \
            }                                                                                                                                         \
            else                                                                                                                                      \
            {                                                                                                                                         \
                /* Overlap okay, refining results */                                                                                                  \
                final_work_time = work_time;                                                                                                          \
                work = updated_overlap_status(params, &overlap_status, total_time, ref_time + stdev, true, work);                                     \
                OVERLAP_DEBUG(params, "Overlap okay, refining results with %" PRId64 " units\n", work);                                               \
            }                                                                                                                                         \
        }                                                                                                                                             \
                                                                                                                                                      \
        if (overlap_status.max_valid_overlap_work_units != -1 && overlap_status.min_invalid_overlap_work_units != -1)                                 \
        {                                                                                                                                             \
            int64_t diff = overlap_status.min_invalid_overlap_work_units - overlap_status.max_valid_overlap_work_units;                               \
            int64_t max_diff = overlap_status.min_invalid_overlap_work_units * params->overlap_threshold / 100;                                       \
            if (diff <= max_diff)                                                                                                                     \
            {                                                                                                                                         \
                /* The difference between the min and max is less than (params->overlap_threshold)% of the max, we stop. */                           \
                OVERLAP_DEBUG(params, "Less than %d%% difference between min (%" PRId64 ") and max (%" PRId64 "), we are done\n",                     \
                              params->overlap_threshold, overlap_status.max_valid_overlap_work_units, overlap_status.min_invalid_overlap_work_units); \
                work = -1;                                                                                                                            \
            }                                                                                                                                         \
        }                                                                                                                                             \
    }

#define TDM_COMPUTE_OVERLAP                                                                                                             \
    if (params->world_rank == 0)                                                                                                        \
    {                                                                                                                                   \
        final_work_time /= n_iters;                                                                                                     \
        if (overlap != 100)                                                                                                             \
        {                                                                                                                               \
            GET_OVERLAP(overlap, ref_time, final_work_time);                                                                            \
            overlap *= 100; /* Convert to pourcentage */                                                                                \
            if (overlap > 100)                                                                                                          \
                overlap = 100; /* This is possible when more work than the initial estimated time is injected because of variability */ \
        }                                                                                                                               \
        fprintf(stdout, "Data size exchanged per rank: %" PRIu64 " bytes\n", n_elts * 8);                                               \
        fprintf(stdout, "Injected work time: %f milli-seconds\n", final_work_time);                                                     \
        fprintf(stdout, "Reference time: %f milli-seconds (stdev: %f)\n", ref_time, stdev);                                             \
        fprintf(stdout, "Overlap: %.0f %%\n", overlap);                                                                                 \
    }

#endif // OVERLAP_TDM_H_