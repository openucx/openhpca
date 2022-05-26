# 

SHELL = /bin/sh
.IGNORE:

# System-specific settings
CC =		mpicc
CCFLAGS =	-O2
LINK =		mpicc
LINKFLAGS =	

# Compilation rules

.c.o:
	$(CC) $(CCFLAGS) -c $<

OBJS = mpi_overhead.o

mpi_overhead: $(OBJS) Makefile
	$(LINK) $(LINKFLAGS) -o mpi_overhead $(OBJS)

clean:
	rm -rf *.o mpi_overhead
