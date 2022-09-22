//
// Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gvallee/go_util/pkg/util"
	"github.com/openucx/openhpca/tools/internal/pkg/config"
	"github.com/openucx/openhpca/tools/internal/pkg/report"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	help := flag.Bool("h", false, "Help message")

	flag.Parse()

	if *help {
		filename := filepath.Base(os.Args[0])
		fmt.Printf("%s generates a report after running openHPCA benchmarks\n", filename)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFile := util.OpenLogFile("openhpca", "report")
	defer logFile.Close()
	if *verbose {
		multiWriters := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multiWriters)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	_, filename, _, _ := runtime.Caller(0)
	basedir := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	cfg := new(config.Data)
	cfg.Basedir = basedir
	cfg.BinName = filename

	// Load the configuration
	err := cfg.Load()
	if err != nil {
		fmt.Printf("Unable to load OpenHPCA configuration: %s\n", err)
		os.Exit(1)
	}

	runDir := cfg.GetRunDir()
	if !util.PathExists(runDir) {
		fmt.Printf("ERROR: run directory %s not found\n", runDir)
		os.Exit(1)
	}

	err = report.Generate(cfg)
	if err != nil {
		fmt.Printf("ERROR: unable to generate the report: %s\n", err)
		os.Exit(1)
	}
}