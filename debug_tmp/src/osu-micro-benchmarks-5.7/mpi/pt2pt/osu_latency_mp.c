#define BENCHMARK "OSU MPI%s Multi-process Latency Test"
/*
 * Copyright (C) 2002-2021 the Network-Based Computing Laboratory
 * (NBCL), The Ohio State University. 
 *
 * Contact: Dr. D. K. Panda (panda@cse.ohio-state.edu)
 *
 * For detailed copyright and licensing information, please refer to the
 * copyright file COPYRIGHT in the top level OMB directory.
 */

#include <osu_util_mpi.h>

void communicate(int myid); 

int main(int argc, char *argv[])
{
    int numprocs = 0, myid = 0;
    int num_processes_sender = 0;
    int i = 0;
    int po_ret = 0;
    int is_child = 0;
    
    pid_t sr_processes[MAX_NUM_PROCESSES];

    options.bench = PT2PT;
    options.subtype = LAT_MP;

    set_header(HEADER);
    set_benchmark_name("osu_latency_mp");

    po_ret = process_options(argc, argv);

    if (PO_OKAY == po_ret && NONE != options.accel) {
        if (init_accel()) {
            fprintf(stderr, "Error initializing device\n");
            exit(EXIT_FAILURE);
        }
    }

    MPI_CHECK(MPI_Init(&argc, &argv));
    MPI_CHECK(MPI_Comm_size(MPI_COMM_WORLD, &numprocs));
    MPI_CHECK(MPI_Comm_rank(MPI_COMM_WORLD, &myid));

    if (0 == myid) {
        switch (po_ret) {
            case PO_CUDA_NOT_AVAIL:
                fprintf(stderr, "CUDA support not available.\n");
                break;
            case PO_OPENACC_NOT_AVAIL:
                fprintf(stderr, "OPENACC support not available.\n");
                break;
            case PO_HELP_MESSAGE:
                print_help_message(myid);
                break;
            case PO_BAD_USAGE:
                print_bad_usage_message(myid);
                break;
            case PO_VERSION_MESSAGE:
                print_version_message(myid);
                MPI_CHECK(MPI_Finalize());
                exit(EXIT_SUCCESS);
            case PO_OKAY:
                break;
        }
    }

    switch (po_ret) {
        case PO_CUDA_NOT_AVAIL:
        case PO_OPENACC_NOT_AVAIL:
        case PO_BAD_USAGE:
            MPI_CHECK(MPI_Finalize());
            exit(EXIT_FAILURE);
        case PO_HELP_MESSAGE:
        case PO_VERSION_MESSAGE:
            MPI_CHECK(MPI_Finalize());
            exit(EXIT_SUCCESS);
        case PO_OKAY:
            break;
    }

    if (numprocs != 2) {
        if (myid == 0) {
            fprintf(stderr, "This test requires exactly two processes\n");
        }

        MPI_CHECK(MPI_Finalize());
        exit(EXIT_FAILURE);
    }

    if (options.sender_processes != -1) {
        num_processes_sender = options.sender_processes;
    }

    print_header(myid, LAT_MP);
    
    if (myid == 0) {
        fprintf(stdout, "# Number of forked processes in sender: %d\n",
                num_processes_sender); 
        fprintf(stdout, "# Number of forked processes in receiver: %d\n",
                options.num_processes );
        fprintf(stdout, "%-*s%*s\n", 10, "# Size", FIELD_WIDTH,
                "Latency (us)");
        fflush(stdout);
        
        for (i = 0; i < num_processes_sender; i++) {
            sr_processes[i] = fork();
            if (sr_processes[i] == 0) {
                is_child = 1;
                break;
            }
        }

        if (is_child == 0) {
            communicate(myid);      
        } else {
            sleep(CHILD_SLEEP_SECONDS);
        }
    } else {
        for (i = 0; i < options.num_threads; i++) {
            sr_processes[i] = fork();
            if (sr_processes[i] == 0) {
                is_child = 1;
                break;
            }
        }
        if (is_child == 0) {
            communicate(myid);
        } else {
            sleep(CHILD_SLEEP_SECONDS);
        }
    }
    
    if (is_child == 0) {
        MPI_CHECK(MPI_Finalize());
        if (NONE != options.accel) {
            if (cleanup_accel()) {
                fprintf(stderr, "Error cleaning up device\n");
                exit(EXIT_FAILURE);
            }
        }
    }

    return EXIT_SUCCESS;
}

void communicate(int myid)
{
    /* Latency test */
    double t_start = 0.0, t_end = 0.0;
    int size = 0, i = 0;
    char *s_buf, *r_buf;
    MPI_Status reqstat;
    
    if (allocate_memory_pt2pt(&s_buf, &r_buf, myid)) {
        /* Error allocating memory */
        MPI_CHECK(MPI_Finalize());
        exit(EXIT_FAILURE);
    }
    
    for (size = options.min_message_size; size <= options.max_message_size; size = (size ? size * 2 : 1)) {
        set_buffer_pt2pt(s_buf, myid, options.accel, 'a', size);
        set_buffer_pt2pt(r_buf, myid, options.accel, 'b', size);

        if (size > LARGE_MESSAGE_SIZE) {
            options.iterations = options.iterations_large;
            options.skip = options.skip_large;
        }

        MPI_CHECK(MPI_Barrier(MPI_COMM_WORLD));

        if (myid == 0) {
            for (i = 0; i < options.iterations + options.skip; i++) {
                if (i == options.skip) {
                    t_start = MPI_Wtime();
                }
                MPI_CHECK(MPI_Send(s_buf, size, MPI_CHAR, 1, 1, 
                        MPI_COMM_WORLD));
                MPI_CHECK(MPI_Recv(r_buf, size, MPI_CHAR, 1, 1, 
                        MPI_COMM_WORLD, &reqstat));
            }
            t_end = MPI_Wtime();
        } else if (myid == 1) {
            for (i = 0; i < options.iterations + options.skip; i++) {
                MPI_CHECK(MPI_Recv(r_buf, size, MPI_CHAR, 0, 1, 
                        MPI_COMM_WORLD, &reqstat));
                MPI_CHECK(MPI_Send(s_buf, size, MPI_CHAR, 0, 1, 
                        MPI_COMM_WORLD));
            }
        }
        if (myid == 0) {
            double latency = (t_end - t_start) * 1e6 / (2.0 * options.iterations);
            fprintf(stdout, "%-*d%*.*f\n", 10, size, FIELD_WIDTH, 
                    FLOAT_PRECISION, latency);
            fflush(stdout);
        }
    }
    free_memory(s_buf, r_buf, myid);
}

/* vi: set sw=4 sts=4 tw=80: */
