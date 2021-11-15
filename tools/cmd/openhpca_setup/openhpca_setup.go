//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
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
	"github.com/gvallee/openhpca/tools/internal/pkg/config"
)

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	help := flag.Bool("h", false, "Help message")
	init := flag.Bool("init-workspace", false, "Initialize a default workspace")

	flag.Parse()

	if *help {
		filename := filepath.Base(os.Args[0])
		fmt.Printf("%s setup all the software components for openHPCA\n", filename)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFile := util.OpenLogFile("openhpca", "setup")
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

	if *init {
		err := cfg.InitWorkspace()
		if err != nil {
			fmt.Printf("Unable to create a workspace: %s\n", err)
			os.Exit(1)
		}

		fmt.Println("A default workspace configuration has been created.")
		fmt.Printf("To customize your workspace, please edit the %s configuration file\n", cfg.WP.ConfigFile)
	}

	// Load the configuration
	err := cfg.Load()
	if err != nil {
		fmt.Printf("Unable to load OpenHPCA configuration: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Installing benchmarks, please wait...")
	err = cfg.Compile()
	if err != nil {
		fmt.Printf("Unable to compile all required software: %s\n", err)
		os.Exit(1)
	}

	cfg.DetectInstalledBenchmarks()
	cfg.Display()
}
