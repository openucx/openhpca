CC=cc
#CC=mpicc
#CFLAGS=-Wall -g -Wstrict-prototypes -Wmissing-prototypes -Wundef -Wpointer-arith -Wbad-function-cast
CFLAGS=-O -g

all: msgrate

rmo:
	rm *.o

msgrate.o: msgrate.c

msgrate: msgrate.o
	$(CC) -o $@ $^

clean:
	rm -f *.o msgrate