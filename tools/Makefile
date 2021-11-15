# Copyright (c) 2020-2021 NVIDIA CORPORATION. All rights reserved.

.PHONY: openhpca_setup openhpca_run webui

all: openhpca_setup openhpca_run webui

webui:
	cd cmd/webui; go build webui.go

openhpca_run:
	cd cmd/openhpca_run; go build openhpca_run.go

openhpca_setup:
	cd cmd/openhpca_setup; go build openhpca_setup.go

clean:
	@rm -f cmd/openhpca_setup/openhpca_setup
	@rm -f cmd/openhpca_run/openhpca_run
	@rm -f cmd/webui/webui
