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
	"log"
	"os"
	"path/filepath"

	"github.com/gvallee/go_util/pkg/util"
	"github.com/openucx/openhpca/tools/internal/pkg/webui"
)

func main() {
	help := flag.Bool("h", false, "Help message")
	port := flag.Int("port", webui.DefaultPort, "Port on which to start the WebUI")
	verbose := flag.Bool("v", false, "Enable verbose mode")

	flag.Parse()

	cmdName := filepath.Base(os.Args[0])
	if *help {
		fmt.Printf("%s starts a Web-based user interface", cmdName)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFile := util.OpenLogFile("openhpca", cmdName)
	defer logFile.Close()
	nultiWriters := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(nultiWriters)

	cfg, err := webui.Init(*verbose)
	if err != nil {
		fmt.Printf("ERROR: initialization of the webUI failed: %s", err)
		os.Exit(1)
	}
	cfg.Port = *port

	server, err := cfg.Start()
	if err != nil {
		fmt.Printf("ERROR: WebUI faced an internal error: %s\n", err)
		os.Exit(1)
	}

	server.Wait()
}
