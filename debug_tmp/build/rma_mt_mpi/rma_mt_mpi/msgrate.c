/* -*- C -*-
 *
 * Copyright 2006 & 2016 Sandia Corporation. Under the terms of Contract
 * DE-AC04-94AL85000 with Sandia Corporation, the U.S. Government
 * retains certain rights in this software.
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, 
 * Boston, MA  02110-1301, USA.
 */

#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <pthread.h>
/* constants */
const int magic_tag = 1;

/* configuration parameters - setable by command line arguments */
int npeers = 6;
int niters = 4096;
int nmsgs = 128;
int nbytes = 8;
int cache_size = (8 * 1024 * 1024 / sizeof(int));
int ppn = -1;
int machine_output = 0;
int threads = 0;
int rma_mode = 0;
int rma_op   = 0;
/* globals */
int *send_peers;
int *recv_peers;
int *cache_buf;
char *send_buf;
char *recv_buf;
MPI_Request *reqs;

int rank = -1;
int world_size = -1;

typedef struct {
    int thread_id;
    int iters;
    MPI_Request *local_reqs;
    int rank;
    char *send_buf;
//  MPI_Comm *comm;
} thread_input;

static void
abort_app(const char *msg)
{
    perror(msg);
    MPI_Abort(MPI_COMM_WORLD, 1);
}


static void
cache_invalidate(void)
{
    int i;

    cache_buf[0] = 1;
    for (i = 1 ; i < cache_size ; ++i) {
        cache_buf[i] = cache_buf[i - 1];
    }
}


static inline double
timer(void)
{
    return MPI_Wtime();
}


void
display_result(const char *test, const double result)
{
    if (0 == rank) {
        if (machine_output) {
            printf("%.2f ", result);
        } else {
            printf("%10s: %.2f\n", test, result);
        }
    }
}


/*********************Start Time FUNCS*************************/

double* thread_etimes;

double find_max(){
  double max = 0;
  int i;
  for (i = 0; i < threads; i++)
    if(max < thread_etimes[i]) max=thread_etimes[i];
  //printf("Max = %f\n", max);
  return max;
}

/*********************End Time FUNCS***************************/

volatile int count = 0;
pthread_mutex_t cntlk;
pthread_cond_t cntcond;

/*********************Start RMA FUNCS**************************/
MPI_Win win;
MPI_Win win2; // 

MPI_Group send_comm_group, send_group;
MPI_Group recv_comm_group, recv_group;

void setupwindow(int single_dir)
{
    MPI_Win_create(recv_buf,
                   npeers * nmsgs * nbytes,
                   1,
                   MPI_INFO_NULL,
                   MPI_COMM_WORLD,
                   &win); 

    if(rma_mode == 2)
    {
        if(single_dir)
        {
            int dest = (rank + world_size/2) % world_size;
            MPI_Comm_group(MPI_COMM_WORLD, &(recv_comm_group));
            MPI_Group_incl(recv_comm_group, 1, &dest, &(recv_group));
            MPI_Comm_group(MPI_COMM_WORLD, &(send_comm_group));
            MPI_Group_incl(send_comm_group, 1, &dest, &(send_group));
        }
        else
        {
            MPI_Comm_group(MPI_COMM_WORLD, &(recv_comm_group));
            MPI_Group_incl(recv_comm_group, npeers, recv_peers, &(recv_group));
            MPI_Comm_group(MPI_COMM_WORLD, &(send_comm_group));
            MPI_Group_incl(send_comm_group, npeers, send_peers, &(send_group));
        }
    }
}


void starttransfer(int single_dir)
{
    int i;
    if(rma_mode == 0) {
        MPI_Win_fence(0, win);
    } else if (rma_mode == 1) {
        if(single_dir)
        {
            MPI_Win_lock(MPI_LOCK_SHARED,(rank+world_size/2)%world_size,0,win);
        }
        else 
        {
            for(i = 0; i < npeers; i++)
            {
                MPI_Win_lock(MPI_LOCK_SHARED,send_peers[i],0,win);
            }
        }
    } else if (rma_mode == 2) {
        MPI_Win_post(recv_group,0,win);
        MPI_Win_start(send_group,0,win);
    } else if (rma_mode == 3) {
        MPI_Win_lock_all(0,win);
    }
}

void transfer(int offset, int dest)
{
    if(rma_op)
    {
        MPI_Put(send_buf+offset, nbytes, MPI_CHAR, dest, offset, nbytes, MPI_CHAR, win); 
    }
    else
    {
        MPI_Get(send_buf+offset, nbytes, MPI_CHAR, dest, offset, nbytes, MPI_CHAR, win);
    }
}

void endtransfer(int single_dir)
{
    int i;
    if(rma_mode == 0) {
        MPI_Win_fence(0, win);
    } else if (rma_mode == 1) {
        if(single_dir)
        {
            MPI_Win_unlock((rank+world_size/2)%world_size,win);
        }
        else
        {
            for(i = 0; (i < npeers); i++)
            {
                MPI_Win_unlock(send_peers[i],win);
            }
        }
    } else if (rma_mode == 2) {
        MPI_Win_complete(win);
        MPI_Win_wait(win);
    }
}

void destroywindow()
{
     MPI_Win_fence(0, win);
     MPI_Win_free(&win);
     if (rma_mode == 2)
     {
        MPI_Group_free(&send_comm_group);
        MPI_Group_free(&send_group);
        MPI_Group_free(&recv_comm_group);
        MPI_Group_free(&recv_group);
     }
}
/***********************End RMA FUNCS**************************/


void *sendmsg(void * input)
{
    int nreqs= 0;
    int i, j;
    thread_input *t_info = (thread_input *)input;

    for (i = 0 ; i < niters ; ++i) 
    {
        pthread_mutex_lock(&cntlk);
        count++;
        pthread_cond_wait(&cntcond, &cntlk);
        pthread_mutex_unlock(&cntlk);

        for (j= 0; j < t_info->iters; j++)
        {
            transfer(((t_info->thread_id * t_info->iters * nbytes) + (nbytes * j)), t_info->rank + (world_size / 2));
        }

        pthread_mutex_lock(&cntlk);
        count--;
        pthread_cond_wait(&cntcond, &cntlk);
        pthread_mutex_unlock(&cntlk);
    }
    //printf("Thread %d finished!\n", t_info->thread_id);
    return NULL;
}

void
test_one_way(void)
{
    int i, k, nreqs;
    double tmp, total = 0;
    double stt, ttt=0;
    setupwindow(1);

    MPI_Barrier(MPI_COMM_WORLD);
    /*printf("start one way\n");
    fflush(stderr);*/
    thread_input *info;
    info = (thread_input *) malloc(threads*sizeof(thread_input)*2);
    pthread_t *pthreads;
    pthreads = (pthread_t*) malloc(threads*sizeof(pthreads)*5);
    reqs = malloc(sizeof(MPI_Request) * 2 * nmsgs * npeers * threads);
    char *local_sendbuf = malloc(npeers * nmsgs * nbytes);
    if (!(world_size % 2 == 1 && rank == (world_size - 1))) {
        if (rank < world_size / 2) {
            int thread_count = 0;
            for (thread_count = 0; thread_count < threads; thread_count++){
                info[thread_count].thread_id = thread_count;
                info[thread_count].iters = nmsgs/threads;
                info[thread_count].local_reqs = &reqs[(nmsgs/threads)*thread_count];
                info[thread_count].rank = rank;
                pthread_create(&pthreads[thread_count], NULL, sendmsg, (void *)&info[thread_count]);
            }
            for (i = 0 ; i < niters ; ++i) {
                cache_invalidate();
                MPI_Barrier(MPI_COMM_WORLD);
                //nreqs = 0;
                while (count != thread_count) {
                }
                pthread_mutex_lock(&cntlk);
                tmp = timer();
                starttransfer(1);
                pthread_cond_broadcast(&cntcond);
                pthread_mutex_unlock(&cntlk);

                while (count != 0) {
                }
                pthread_mutex_lock(&cntlk);
                endtransfer(1);
                total += (timer() - tmp);
                pthread_cond_broadcast(&cntcond);
                pthread_mutex_unlock(&cntlk);
                 
                //MPI_Barrier(MPI_COMM_WORLD);
                //total += (timer() - tmp);
            }
        } else {
            for (i = 0 ; i < niters ; ++i) {
                cache_invalidate();
                MPI_Barrier(MPI_COMM_WORLD);
                tmp = timer();
                starttransfer(1);
                endtransfer(1);
                //MPI_Barrier(MPI_COMM_WORLD);
                total += (timer() - tmp);
            }
        }
        MPI_Allreduce(&total, &tmp, 1, MPI_DOUBLE, MPI_SUM, MPI_COMM_WORLD);
        display_result("single direction ", (niters * nmsgs * ((world_size)/2)) / (tmp / world_size));
        //display_result("single dir-TO    ", (niters * nmsgs * ((world_size)/2)) / ((tmp - ttt) / world_size));
    }
    free(info);
    free(pthreads);
    free(reqs);
    free(local_sendbuf);
    destroywindow();
    MPI_Barrier(MPI_COMM_WORLD);
}

void *all_run(void * input)
{
    int nreqs= 0;
    int i,j,k;
    thread_input *t_info = (thread_input *)input;
    for (i = 0 ; i < niters ; ++i) {
        pthread_mutex_lock(&cntlk);
        count++;
        pthread_cond_wait(&cntcond, &cntlk);
        pthread_mutex_unlock(&cntlk);
        for (j = 0 ; j < npeers ; ++j) {
            for (k = 0 ; k < t_info->iters ; ++k) {
                  transfer(nbytes * (k + t_info->thread_id * t_info->iters + j * nmsgs), send_peers[npeers - j - 1]);
            }
        }
        pthread_mutex_lock(&cntlk);
        count--;
        pthread_cond_wait(&cntcond, &cntlk);
        pthread_mutex_unlock(&cntlk);
    }
    return NULL;
}

void
test_all(void)
{
    int i, k, nreqs;
    double tmp, total = 0;
    double stt, ttt=0;
    setupwindow(0);
    MPI_Barrier(MPI_COMM_WORLD);

    thread_input *info;
    info = (thread_input *) malloc(threads*sizeof(thread_input)*2);
    pthread_t *pthreads;
    pthreads = (pthread_t*) malloc(threads*sizeof(pthreads)*5);
    reqs = malloc(sizeof(MPI_Request) * 2 * nmsgs * npeers * threads);
    char *local_sendbuf = malloc(npeers * nmsgs * nbytes);
    int thread_count = 0;
    for (thread_count = 0; thread_count < threads; thread_count++){
        info[thread_count].thread_id = thread_count;
        info[thread_count].iters = nmsgs/threads;
        info[thread_count].local_reqs = &reqs[2*(nmsgs/threads)*npeers*thread_count];
        info[thread_count].rank = rank;
        pthread_create(&pthreads[thread_count], NULL, all_run, (void *)&info[thread_count]);
    }    
    for (i = 0 ; i < niters ; ++i) {
        cache_invalidate();
        MPI_Barrier(MPI_COMM_WORLD);

        while (count != thread_count) {
        }
        pthread_mutex_lock(&cntlk);
        tmp = timer();
        starttransfer(0);
        pthread_cond_broadcast(&cntcond);
        pthread_mutex_unlock(&cntlk);

        while (count != 0) {
        }
        pthread_mutex_lock(&cntlk);
        endtransfer(0);
        //MPI_Barrier(MPI_COMM_WORLD);
        total += (timer() - tmp);
        pthread_cond_broadcast(&cntcond);
        pthread_mutex_unlock(&cntlk);
    }
    MPI_Allreduce(&total, &tmp, 1, MPI_DOUBLE, MPI_SUM, MPI_COMM_WORLD);
    display_result("halo-exchange    ", (niters * npeers * nmsgs * 2) / (tmp / world_size));
    free(info);
    free(pthreads);
    free(reqs);
    free(local_sendbuf);
    MPI_Barrier(MPI_COMM_WORLD);
    destroywindow();
}


void
usage(void)
{
    fprintf(stderr, "Usage: msgrate -n <ppn> [OPTION]...\n\n");
    fprintf(stderr, "  -h           Display this help message and exit\n");
    fprintf(stderr, "  -p <num>     Number of peers used in communication\n");
    fprintf(stderr, "  -i <num>     Number of iterations per test\n");
    fprintf(stderr, "  -m <num>     Number of messages per peer per iteration\n");
    fprintf(stderr, "  -s <size>    Number of bytes per message\n");
    fprintf(stderr, "  -c <size>    Cache size in bytes\n");
    fprintf(stderr, "  -n <ppn>     Number of procs per node\n");
    fprintf(stderr, "  -o           Format output to be machine readable\n");
    fprintf(stderr, "  -r           RMA Sych: 0-Fence, 1-Lock Unlock, 2-PSWC\n");
    fprintf(stderr, "  -u           RMA Op: 0-Get, 1-Put\n");
    fprintf(stderr, "\nReport bugs to <bwbarre@sandia.gov>\n");
}


int
main(int argc, char *argv[])
{
    int start_err = 0;
    int i;
    int prov;
    

    MPI_Init_thread(&argc, &argv, MPI_THREAD_MULTIPLE, &prov);
    if (prov != MPI_THREAD_MULTIPLE)
       abort();
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &world_size);


    pthread_mutex_init(&cntlk, NULL);
    pthread_cond_init(&cntcond, NULL);

    /* root handles arguments and bcasts answers */
    if (0 == rank) {
        int ch;
        while (start_err != 1 && 
               (ch = getopt(argc, argv, "p:i:m:s:c:n:o:t:r:u:h")) != -1) {
            switch (ch) {
            case 'p':
                npeers = atoi(optarg);
                break;
            case 'i':
                niters = atoi(optarg);
                break;
            case 'm':
                nmsgs = atoi(optarg);
                break;
            case 's':
                nbytes = atoi(optarg);
                break;
            case 'c':
                cache_size = atoi(optarg) / sizeof(int);
                break;
            case 'n':
                ppn = atoi(optarg);
                break;
            case 'o':
                machine_output = 1;
                break;
            case 't':
                threads = atoi(optarg);
                break;
            case 'r':
                rma_mode = atoi(optarg);
                break;
            case 'u':
                rma_op = atoi(optarg);
                break;
            case 'h':
            case '?':
            default:
                start_err = 1;
                usage();
            }
        }

        /* sanity check */
        if (start_err != 1) {
            if (world_size < 3) {
                fprintf(stderr, "Error: At least three processes are required\n");
                start_err = 1;
            } else if (world_size <= npeers) {
                fprintf(stderr, "Error: job size (%d) <= number of peers (%d)\n",
                        world_size, npeers);
                start_err = 1;
            } else if (ppn < 1) {
                fprintf(stderr, "Error: must specify process per node (-n #)\n");
                start_err = 1;
            } else if ((double)world_size / (double)ppn <= (double)npeers) {
                fprintf(stderr, "Error: node count <= number of peers %i / %i <= %i\n",world_size,ppn,npeers);
                start_err = 1;
            } else if (world_size % 2 == 1) {
                fprintf(stderr, "Error: node count of %d isn't even.\n", world_size);
                start_err = 1;
            }
        }
    }

    /* broadcast results */
    MPI_Bcast(&start_err, 1, MPI_INT, 0, MPI_COMM_WORLD);
    if (0 != start_err) {
        MPI_Finalize();
        exit(1);
    }
    MPI_Bcast(&npeers, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&niters, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&nmsgs, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&nbytes, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&cache_size, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&ppn, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&threads, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&rma_mode, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&rma_op, 1, MPI_INT, 0, MPI_COMM_WORLD);
    if (0 == rank) {
        if (!machine_output) {
            printf("job size         : %d\n", world_size);
            printf("npeers           : %d\n", npeers);
            printf("niters           : %d\n", niters);
            printf("nmsgs            : %d\n", nmsgs);
            printf("nbytes           : %d\n", nbytes);
            printf("cache size       : %d\n", cache_size * (int)sizeof(int));
            printf("ppn              : %d\n", ppn);
            printf("threads          : %d\n", threads);
            printf("RMA Mode         : ");
            switch(rma_mode)
            {
                case 0:
                    printf("fence\n");
                    break;
                case 1:
                    printf("lock/unlock\n");
                    break;
                case 2:
                    printf("post start wait complete\n");
                    break;
            }
            printf("RMA Op           : ");
            switch(rma_op)
            {
                case 0:
                    printf("Get\n");
                    break;
                case 1:
                    printf("Put\n");
                    break;
            }
        } else {
            printf("%d %d %d %d %d %d %d %d %d %d ", 
                   world_size, npeers, niters, nmsgs, nbytes,
                   cache_size * (int)sizeof(int), ppn, threads, rma_mode, rma_op);
        }
    }

    /* allocate buffers */
    send_peers = malloc(sizeof(int) * npeers);
    if (NULL == send_peers) abort_app("malloc");
    recv_peers = malloc(sizeof(int) * npeers);
    if (NULL == recv_peers) abort_app("malloc");
    cache_buf = malloc(sizeof(int) * cache_size);
    if (NULL == cache_buf) abort_app("malloc");
    send_buf = malloc(npeers * nmsgs * nbytes);
    if (NULL == send_buf) abort_app("malloc");
    thread_etimes = malloc(sizeof(double)*threads);    
    //bad_buf = malloc(npeers * nmsgs * nbytes);
    //if (NULL == send_buf) abort_app("malloc");
    
    recv_buf = malloc(npeers * nmsgs * nbytes);
    if (NULL == recv_buf) abort_app("malloc");
    reqs = malloc(sizeof(MPI_Request) * 2 * nmsgs * npeers);
    if (NULL == reqs) abort_app("malloc");

    /* calculate peers */
    for (i = 0 ; i < npeers ; ++i) {
        if (i < npeers / 2) {
            send_peers[i] = (rank + world_size + ((i - npeers / 2) * ppn)) % world_size;
        } else {
            send_peers[i] = (rank + world_size + ((i - npeers / 2 + 1) * ppn)) % world_size;
        }
    }
    if (npeers % 2 == 0) {
        /* even */
        for (i = 0 ; i < npeers ; ++i) {
            if (i < npeers / 2) {
                recv_peers[i] = (rank + world_size + ((i - npeers / 2) *ppn)) % world_size;
            } else {
                recv_peers[i] = (rank + world_size + ((i - npeers / 2 + 1) * ppn)) % world_size;
            }
        } 
    } else {
        /* odd */
        for (i = 0 ; i < npeers ; ++i) {
            if (i < npeers / 2 + 1) {
                recv_peers[i] = (rank + world_size + ((i - npeers / 2 - 1) * ppn)) % world_size;
            } else {
                recv_peers[i] = (rank + world_size + ((i - npeers / 2) * ppn)) % world_size;
            }
        }
    }

    /* BWB: FIX ME: trash the free lists / malloc here */

    /* sync, although tests will do this on their own (in theory) */
    MPI_Barrier(MPI_COMM_WORLD);

    /* run tests */
    test_one_way();
    test_all();
    if (rank == 0 && machine_output) printf("\n");
    /* done */
    MPI_Finalize();
    return 0;
}
