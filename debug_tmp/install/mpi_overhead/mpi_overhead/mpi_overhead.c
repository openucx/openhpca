/*****************************************************************************

 Copyright 2006 Sandia Corporation. Under the terms of Contract
 DE-AC04-94AL85000 with Sandia Corporation, the U.S. Government
 retains certain rights in this software.

 This program is free software; you can redistribute it and/or
 modify it under the terms of the GNU General Public License
 as published by the Free Software Foundation; either version 2
 of the License, or (at your option) any later version.

 This program is distributed in the hope that it will be useful,
 but WITHOUT ANY WARRANTY; without even the implied warranty of
 MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 GNU General Public License for more details.

 You should have received a copy of the GNU General Public License
 along with this program; if not, write to the Free Software
 Foundation, Inc., 51 Franklin Street, Fifth Floor, 
 Boston, MA  02110-1301, USA.

 name:		mpi_overhead.c

 purpose:	A benchmark to calculate the host overhead associated with 
 		sending and receiving MPI messages.

*****************************************************************************/
#include <stdio.h>
#include <stddef.h>
#include <stdlib.h>
#include <signal.h>
#include <getopt.h>
#include <mpi.h>

#include <ctype.h>
#include <string.h>

#define AUTO_ITERATIONS       0
#define MAX_ITERATIONS        1000
#define MIN_ITERATIONS        10
#define MAX_INTERVAL          1.0    /* seconds */
#define MIN_DATA_XFER         8      /* bytes */
#define MIN_WORK              1      /* # of work loop iterations */
#define WORK_FACTOR_1         2.0
#define THRESH_FACTOR         1.5
#define BASE_THRESH_FACTOR    1.02

#define OVRCV		      0
#define OVSND		      1
#define OVNOP		      2
#define OVDONE		      3

static struct option mpi_overhead_options[] = {
  { "help",       no_argument,       NULL, 'q' },
  { "verbose",    no_argument,       NULL, 'v' },
  { "recv",       no_argument,       NULL, 'r' },
  { "iterations", required_argument, NULL, 'i' },
  { "msgsize",    required_argument, NULL, 'm' },
  { "thresh",     required_argument, NULL, 't' },
  { "bthresh",    required_argument, NULL, 'b' },
  { "nohdr",      no_argument,       NULL, 'n' },
  { NULL,         0,                 NULL,  0  }
};

/* placing the work variables as globals keeps the compiler from 
   optimizing the work loop away. But this may not work for all 
   compilers, so be careful. */ 
int x;
double y, a = 1.0, b = 1.0;

void show_partials(int label, double iter_t, double base_t)
{
  static int first_time = 1;

  if (first_time) {
    fprintf(stdout, "\
%-8s\
%-13s\
%-13s\n\
", "work", "iter_t", "base_t");
    first_time = 0;
  }

  fprintf(stdout, "\
%-8d\
%-13.3lf\
%-13.3lf\n\
", label, iter_t*1e6, base_t*1e6);
}

void show_results(int verbose, int nohdr, int data_size, int iterations, 
int work, double iter_t, double work_t, double overhead, double base_t)
{
  double availability = (double)100 * ((double)1.0 - overhead / base_t);

  if (!nohdr) {
      fprintf(stdout, "\
%-8s\
%-12s\
%-12s\
%-12s\
%-12s\
%-12s\
%-12s\n\
", "msgsize", "iterations", "iter_t", "work_t", "overhead", "base_t", 
"avail(%)");
  }

  fprintf(stdout, "\
%-8d\
%-12d\
%-12.3lf\
%-12.3lf\
%-12.3lf\
%-12.3lf\
%-12.1lf\n\
", data_size, iterations, iter_t*1e6, work_t*1e6, overhead*1e6, base_t*1e6, 
availability);
}

void usage(char *name) {
  fprintf(stderr, "  Usage:%s \n\
  [-r | --recv] measure receive overhead, default is send\n\
  [-i | --iterations num_iterations] default = autocalculate\n\
  [-m | --msgsize size] default = %d bytes\n\
  [-t | --thresh threshold] default = %f \n\
  [-b | --bthresh base time threshold] default = %f \n\
  [-n | --nohdr] don't print header information, default == off\n\
  [-v | --verbose] print partial results, default == off\n",
  name, MIN_DATA_XFER, THRESH_FACTOR, BASE_THRESH_FACTOR);
}


int main(argc, argv)
int argc;
char **argv;
{
  MPI_Status rstatus, sstatus;
  MPI_Request rrequest, srequest;
  static void *msg = NULL;
  unsigned char *tmsg;
  double start1, time1, base_time, iter_time, work_time, accum_time, overhead, 
    work_factor, threshold, base_threshold;
  int nohdr, verbose, count, size, rank, dest_node, iter, iterations, 
    work, new_work, last_work, data_size, accum_count, continue_flag;
  int error, ch;
  enum {send, recv} direction;
  struct {
      int command;
      int iterations;
  } message;

  /**************************************************************
    Initialize MPI
  **************************************************************/
  if ( MPI_Init( &argc, &argv ) != MPI_SUCCESS ) {
    fprintf(stderr, "Unable to initialize MPI\n");
    exit(0);
  }

  MPI_Comm_size( MPI_COMM_WORLD, &size );
  MPI_Comm_rank( MPI_COMM_WORLD, &rank );

  if ( size % 2 ) {
      if ( rank == 0 ) {
          fprintf(stderr, "ERROR: This program requires # processors be a \
multiple of 2\n");
      }
      MPI_Abort( MPI_COMM_WORLD, 0);
  }

  /**************************************************************
    Parse the Command Args
  **************************************************************/
  error = 0;
  verbose = 0;
  nohdr = 0;
  direction = send;
  iterations = AUTO_ITERATIONS;
  data_size = MIN_DATA_XFER;
  threshold = THRESH_FACTOR;
  base_threshold = BASE_THRESH_FACTOR;
  while ((ch = getopt_long(argc, argv, "qvrni:m:t:b:", mpi_overhead_options, NULL)) != -1) {
    switch (ch) {
      case 'q':
        if (rank == 0) usage(argv[0]);
        error = 1;
        break;
      case 'n':
        nohdr = 1;
        break;
      case 'v':
        verbose = 1;
        break;
      case 'r':
        direction = recv;
        break;
      case 'i':
        if (sscanf(optarg,"%d",&iterations) != 1) {
          fprintf(stderr, "Invalid iterations argument, Exiting\n");
          error = 1;
        }
        break;
      case 'm':
        if (sscanf(optarg,"%d",&data_size) != 1) {
          fprintf(stderr, "Invalid msgsize argument, Exiting\n");
          error = 1;
        }
        break;
      case 't':
        if (sscanf(optarg,"%lf",&threshold) != 1) {
          fprintf(stderr, "Invalid threashold argument, Exiting\n");
          error = 1;
        }
        break;
      case 'b':
        if (sscanf(optarg,"%lf",&base_threshold) != 1) {
          fprintf(stderr, "Invalid base threashold argument, Exiting\n");
          error = 1;
        }
        break;
      default:
        if (rank == 0) usage(argv[0]);
        error = 1;
    }
  }
  argc -= optind;
  argv +- optind;

  if (error) {
    MPI_Finalize();
    exit(0);
  }

  /**************************************************************
    Allocate messaging resources
  **************************************************************/
  if (msg == NULL) {
    if ((msg = malloc(data_size)) == NULL) {
      fprintf(stderr, "Unable to allocate memory\n");
      exit(0);
    }
    for (tmsg = (unsigned char *)msg, count = 0; count < data_size; 
    ++tmsg, ++count)
      *tmsg = count;
  }

  if (!rank && verbose) {
    if (direction == send)
      fprintf(stdout, "Calculating send overhead\n");
    else
      fprintf(stdout, "Calculating receive overhead\n");
    if (iterations != AUTO_ITERATIONS)
      fprintf(stdout, "Using %d iterations per test\n", iterations);
    else
      fprintf(stdout, "Iterations are being calculated automatically\n");
    fprintf(stdout, "Message size is %d bytes\n", data_size);
    fprintf(stdout, "Overhead comparison threshold is %lf (%.1lf%%)\n", 
            threshold, (double)100 * (threshold - (double)1));
    fprintf(stdout, "Base time comparison threshold is %lf (%.1lf%%)\n", 
            base_threshold, (double)100 * (base_threshold - (double)1));
    fprintf(stdout, "Timing resolution is %.2lf uS\n", MPI_Wtick()*1e6);
  }

  /* autocalculate the number of iterations per test */
  if (iterations == AUTO_ITERATIONS) {
    if      (data_size < 64*1024)     iterations = MAX_ITERATIONS;
    else if (data_size < 8*1024*1024) iterations = MAX_ITERATIONS / 10;
    else                              iterations = MAX_ITERATIONS / 100;

    if (iterations < MIN_ITERATIONS) iterations = MIN_ITERATIONS;

    if (!rank && verbose)
      fprintf(stdout, "Using %d iterations per work value\n", iterations);
  }

  MPI_Barrier( MPI_COMM_WORLD );

  /**************************************************************
    Timing Node
  **************************************************************/
  if ( rank < (size / 2) ) {     /* lower half of the rank */
    dest_node = rank + size / 2;
    work = MIN_WORK;
    work_factor = WORK_FACTOR_1;
    accum_time = 0.0;
    accum_count = 0;
    do {
      message.iterations = iterations;
      if (direction == send)
        message.command = OVRCV;
      else
	message.command = OVSND;
      if ( MPI_Send( (int *)&message,
	             2,
		     MPI_INT,
		     dest_node,
		     1,
		     MPI_COMM_WORLD) != MPI_SUCCESS ) {
	fprintf(stderr, "command send failed\n");
	MPI_Abort( MPI_COMM_WORLD, 0);
      }
      MPI_Barrier(MPI_COMM_WORLD);

      start1 = MPI_Wtime();
      if (direction == send) {
        for (iter = 0; iter < iterations; ++iter) {

	  /* This barrier ensures the transfer from the previous 
	     iteration is complete and the recieve side is ready.
	     It will be subtracted from the total time as a 
	     part of the work time calculation. */
	  MPI_Barrier(MPI_COMM_WORLD);
	  
	  /* Send the data to the slave */
          if ( MPI_Isend( msg,
		          data_size,
		          MPI_BYTE,
		          dest_node,
		          2,
		          MPI_COMM_WORLD,
			  &srequest ) != MPI_SUCCESS ) {
	    fprintf(stderr, "Unable to send data\n");
	    MPI_Abort( MPI_COMM_WORLD, 0);
          }

	  /* Work */
	  for (x = 0; x < work; ++x)
	    y = a * (double)x + b;

	  /* wait for the send to complete */
	  MPI_Wait(&srequest, &sstatus);
        }
      }

      else { /* direction == recv */
        for (iter = 0; iter < iterations; ++iter) {

          /* Slave sends data */
          if ( MPI_Irecv( msg,
                          data_size,
                          MPI_BYTE,
	                  dest_node,
	                  2,
	                  MPI_COMM_WORLD,
	                  &rrequest ) != MPI_SUCCESS ) {
	    fprintf(stderr, "Error receiving data\n");
	    MPI_Abort( MPI_COMM_WORLD, 0);
          }

	  MPI_Barrier(MPI_COMM_WORLD);

	  /* Work */
	  for (x = 0; x < work; ++x)
	    y = a * (double)x + b;

	  /* wait for the send to complete */
	  MPI_Wait(&rrequest, &rstatus);
	}
      }
      time1 = (MPI_Wtime() - start1) / iterations;

      if (work == MIN_WORK) 
	base_time = time1;

      last_work = work;
      iter_time = time1;

      /* check to see if we're past the knee of the curve, 
         i.e. time1 is rising */
      if (time1 > (base_time * threshold)) {
        if (verbose)
          show_partials(work, time1, base_time);
	break;
      }

      /* low pass filter the flat part of curve to determine 
         the base time */
      if ( time1 < (base_time * base_threshold) ) {
        accum_time += time1;
        base_time = accum_time / ++accum_count;
      }

      if (verbose)
        show_partials(work, time1, base_time);

      if (work == 0) 
        new_work = 2;
      else
        new_work = work * work_factor;

      if (new_work == work)
        break;
      else
        work = new_work;

    } while(1);

    /* Work includes everything extra required for this benchmark
       i.e. the timer logic and the barrier */
    message.iterations = iterations;
    message.command = OVNOP;
    if ( MPI_Send( (int *)&message,
	           2,
		   MPI_INT,
		   dest_node,
		   1,
		   MPI_COMM_WORLD) != MPI_SUCCESS ) {
      fprintf(stderr, "command send failed\n");
      MPI_Abort( MPI_COMM_WORLD, 0);
    } 
    MPI_Barrier(MPI_COMM_WORLD);

    start1 = MPI_Wtime();
    for (iter = 0; iter < iterations; ++iter) {
      MPI_Barrier(MPI_COMM_WORLD);
      for (x = 0; x < last_work; ++x)
        y = a * (double)x + b;
    }
    work_time = (MPI_Wtime() - start1) / iterations;

    /* Overhead is the iteration time minus the work time */
    overhead = iter_time - work_time;

    if (!rank)
      show_results(verbose, nohdr, data_size, iterations, last_work, 
        iter_time, work_time, overhead, base_time);

    /* terminate the session. */
    message.command = OVDONE;
    if ( MPI_Send( (int *)&message,
                   2,
                   MPI_INT,
	           dest_node,
	           1,
	           MPI_COMM_WORLD) != MPI_SUCCESS ) {
      fprintf(stderr, "command send failed\n");
      MPI_Abort( MPI_COMM_WORLD, 0);
    }
    MPI_Barrier(MPI_COMM_WORLD);

  } /* end of timing node logic */

  /**************************************************************
    Slave Node
  **************************************************************/
  else {
    dest_node = rank - size / 2;

    /* receive data until told otherwise */
    continue_flag = 1;
    do {


      /* get the command */
      if ( MPI_Recv(  (int *)&message,
                      2,
                      MPI_INT,
	              dest_node,
	              1,
	              MPI_COMM_WORLD,
	              &rstatus ) != MPI_SUCCESS ) {
	fprintf(stderr, "Error receiving command\n");
	MPI_Abort( MPI_COMM_WORLD, 0);
      }
      MPI_Barrier(MPI_COMM_WORLD);

      switch (message.command) {
        case OVRCV: /* receive the message */
	  for (iter = 0; iter < message.iterations; ++iter) {
	    /* prepost the receive to avoid an unexpected message */
            if ( MPI_Irecv( msg,
                            data_size,
                            MPI_BYTE,
	                    dest_node,
	                    2,
	                    MPI_COMM_WORLD,
	                    &rrequest ) != MPI_SUCCESS ) {
	      fprintf(stderr, "Error receiving data from the client\n");
	      MPI_Abort( MPI_COMM_WORLD, 0);
            }
	    MPI_Barrier(MPI_COMM_WORLD);
            MPI_Wait(&rrequest, &rstatus);
	  }
	  break;

        case OVSND: /* send the message */
	  for (iter = 0; iter < message.iterations; ++iter) {
	    MPI_Barrier(MPI_COMM_WORLD);
            if ( MPI_Send( msg,
                           data_size,
                           MPI_BYTE,
	                   dest_node,
	                   2,
	                   MPI_COMM_WORLD) != MPI_SUCCESS ) {
	      fprintf(stderr, "Error sending data to the client\n");
	      MPI_Abort( MPI_COMM_WORLD, 0);
            }
	  }
	  break;
  
        case OVNOP: /* just the barrier in the loop */
	  for (iter = 0; iter < message.iterations; ++iter) {
	    MPI_Barrier(MPI_COMM_WORLD);
	  }
	  break;

        case OVDONE: /* we're all done */
          continue_flag = 0;
	  break;
      }

    } while (continue_flag);

  } /* end of slave node */

  free (msg);
  MPI_Finalize();

  if (!rank && verbose)
    fprintf(stdout, "%s: done\n", argv[0]);

  exit(0);
}
