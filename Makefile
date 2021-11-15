.PHONY: check update tools init install

.SILENT:
all: check tools
	@echo "Done"

GOCMD := $(shell command -v go 2>/dev/null)
ifndef GOCMD
tools:
	@echo "Go not installed; skipping tools' compilation"
else
tools:
	cd tools && make;
endif

check:

init:
	./tools/cmd/openhpca_setup/openhpca_setup

install: check update tools init

update:
	git submodule init
	git submodule update --remote

clean:
	cd SMB/src/mpi_overhead; make clean
	cd SMB/src/msgrate; make clean
	cd tools; make clean
	cd src/overlap; make clean
