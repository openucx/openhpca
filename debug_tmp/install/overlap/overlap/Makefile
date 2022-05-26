#
# Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
#
# See LICENSE.txt for license information
#

CFLAGS=-Wall -std=c99

all: overlap_ialltoall \
	overlap_ialltoallv \
	overlap_ireduce \
	overlap_iallreduce \
	overlap_ibcast \
	overlap_ibarrier \
	overlap_igather \
	overlap_igatherv \
	overlap_iallgather \
	overlap_iallgatherv

overlap_igather: overlap_igather.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_igather overlap_igather.c -lm

overlap_igatherv: overlap_igatherv.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_igatherv overlap_igatherv.c -lm

overlap_iallgather: overlap_iallgather.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_iallgather overlap_iallgather.c -lm

overlap_iallgatherv: overlap_iallgatherv.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_iallgatherv overlap_iallgatherv.c -lm

overlap_ialltoall: overlap_ialltoall.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_ialltoall overlap_ialltoall.c -lm

overlap_ialltoallv: overlap_ialltoallv.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_ialltoallv overlap_ialltoallv.c -lm

overlap_ireduce: overlap_ireduce.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_ireduce overlap_ireduce.c -lm

overlap_iallreduce: overlap_iallreduce.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_iallreduce overlap_iallreduce.c -lm

overlap_ibcast: overlap_ibcast.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_ibcast overlap_ibcast.c -lm

overlap_ibarrier: overlap_ibarrier.c overlap.h overlap_tdm.h overlap_ddm.h
	mpicc -Wno-format-zero-length ${CFLAGS} -o overlap_ibarrier overlap_ibarrier.c -lm

clean:
	@rm -f overlap_ireduce
	@rm -f overlap_iallreduce
	@rm -f overlap_ialltoall
	@rm -f overlap_ialltoallv
	@rm -f overlap_igather
	@rm -f overlap_igatherv
	@rm -f overlap_iallgather
	@rm -f overlap_iallgatherv
	@rm -f overlap_ibcast
	@rm -f overlap_ibarrier
