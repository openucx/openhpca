//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
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

	"github.com/gvallee/go_benchmark/pkg/benchmark"
	"github.com/gvallee/go_hpc_jobmgr/pkg/implem"
	"github.com/gvallee/go_software_build/pkg/app"
	"github.com/gvallee/go_util/pkg/util"
	"github.com/gvallee/validation_tool/pkg/experiments"
	"github.com/gvallee/validation_tool/pkg/platform"
	"github.com/openucx/openhpca/tools/internal/pkg/config"
	"github.com/openucx/openhpca/tools/internal/pkg/overlap"
	"github.com/openucx/openhpca/tools/internal/pkg/result"
	"github.com/openucx/openhpca/tools/internal/pkg/smb"
)

func getRunDir(cfg *config.Data) string {
	return filepath.Join(cfg.WP.Basedir, "run")
}

func displayResults(cfg *config.Data) error {
	runDir := getRunDir(cfg)
	resultsStr, err := result.String(runDir)
	if err != nil {
		return err
	}
	fmt.Printf("\nOpenHPCA:\n" + resultsStr)
	resultFile := filepath.Join(cfg.Basedir, "..", result.FileName)
	err = ioutil.WriteFile(resultFile, []byte(resultsStr), result.FilePermission)
	if err != nil {
		return err
	}
	return nil
}

func experimentIsStrictlyPointToPoint(name string) bool {
	switch name {
	case "osu_latency":
		return true
	case "osu_noncontig_mem_latency":
		return true
	case "osu_bw":
		return true
	case "osu_noncontig_mem_bw":
		return true
	case "smb_mpi_overhead":
		return true
	default:
		return false
	}
}

func userSelectedAllBenchmarks(cfg *config.Data) bool {
	if cfg.UserSelection.OsuNoncontigmemSelected &&
		cfg.UserSelection.OsuSelected &&
		cfg.UserSelection.SmbSelected &&
		cfg.UserSelection.OverlapSelected {
		return true
	}
	return false
}

func userSelectedAtLeastOneBenchmark(cfg *config.Data) bool {
	if cfg.UserSelection.OsuNoncontigmemSelected ||
		cfg.UserSelection.OsuSelected ||
		cfg.UserSelection.SmbSelected ||
		cfg.UserSelection.OverlapSelected {
		return true
	}
	return false
}

func selectBenchmarksToRun(cfg *config.Data) map[string]*benchmark.Install {
	var benchmarksToRun map[string]*benchmark.Install
	benchmarksToRun = make(map[string]*benchmark.Install)

	// Check the options to make sure we know what is required
	if userSelectedAllBenchmarks(cfg) {
		cfg.UserSelection.LongRun = true
	}

	if !userSelectedAtLeastOneBenchmark(cfg) && !cfg.UserSelection.LongRun {
		// We need to run all the benchmarks required to get the OpenHPCA metrics
		// We only keep the installed benchmarks that are part of the list of
		// benchmarks required to generate the final metrics
		var osuBenchmarksToRun []app.Info
		installedOSUSubBenchmarks := cfg.InstalledBenchmarks["osu"]
		for _, name := range config.OSURequiredBenchmarks {
			for _, app := range installedOSUSubBenchmarks.SubBenchmarks {
				if app.Name == name {
					osuBenchmarksToRun = append(osuBenchmarksToRun, app)
					break
				}
			}
		}
		benchmarksToRun["osu"] = new(benchmark.Install)
		benchmarksToRun["osu"].SubBenchmarks = osuBenchmarksToRun

		var smbBenchmarksToRun []app.Info
		installedSMBSubBenchmarks := cfg.InstalledBenchmarks["smb"]
		for _, name := range smb.RequiredBenchmarks {
			for _, app := range installedSMBSubBenchmarks.SubBenchmarks {
				if app.Name == name {
					smbBenchmarksToRun = append(smbBenchmarksToRun, app)
					break
				}
			}
		}
		benchmarksToRun["smb"] = new(benchmark.Install)
		benchmarksToRun["smb"].SubBenchmarks = smbBenchmarksToRun

		var overlapBenchmarksToRun []app.Info
		installOverlapSubBenchmarks := cfg.InstalledBenchmarks["overlap"]
		for _, name := range overlap.RequiredBenchmarks {
			for _, app := range installOverlapSubBenchmarks.SubBenchmarks {
				if app.Name == name {
					overlapBenchmarksToRun = append(overlapBenchmarksToRun, app)
					break
				}
			}
		}
		benchmarksToRun["overlap"] = new(benchmark.Install)
		benchmarksToRun["overlap"].SubBenchmarks = overlapBenchmarksToRun
		return benchmarksToRun
	}

	if !cfg.UserSelection.LongRun && userSelectedAtLeastOneBenchmark(cfg) {
		// Get the list of selected benchmarks by the user
		if cfg.UserSelection.OsuSelected {
			var osuBenchmarksToRun []app.Info
			installedOSUSubBenchmarks := cfg.InstalledBenchmarks["osu"]
			osuBenchmarksToRun = append(osuBenchmarksToRun, installedOSUSubBenchmarks.SubBenchmarks...)
			benchmarksToRun["osu"] = new(benchmark.Install)
			benchmarksToRun["osu"].SubBenchmarks = osuBenchmarksToRun
		}

		// At the moment selecting OSU and OSU for non-contiguous memory is the same thing
		if cfg.UserSelection.OsuNoncontigmemSelected {
			var osuBenchmarksToRun []app.Info
			installedOSUSubBenchmarks := cfg.InstalledBenchmarks["osu"]
			osuBenchmarksToRun = append(osuBenchmarksToRun, installedOSUSubBenchmarks.SubBenchmarks...)
			benchmarksToRun["osu"] = new(benchmark.Install)
			benchmarksToRun["osu"].SubBenchmarks = osuBenchmarksToRun
		}

		if cfg.UserSelection.SmbSelected {
			var smbBenchmarksToRun []app.Info
			installedSMBSubBenchmarks := cfg.InstalledBenchmarks["smb"]
			smbBenchmarksToRun = append(smbBenchmarksToRun, installedSMBSubBenchmarks.SubBenchmarks...)
			benchmarksToRun["smb"] = new(benchmark.Install)
			benchmarksToRun["smb"].SubBenchmarks = smbBenchmarksToRun
		}

		if cfg.UserSelection.OverlapSelected {
			var overlapBenchmarksToRun []app.Info
			installOverlapSubBenchmarks := cfg.InstalledBenchmarks["overlap"]
			overlapBenchmarksToRun = append(overlapBenchmarksToRun, installOverlapSubBenchmarks.SubBenchmarks...)
			benchmarksToRun["overlap"] = new(benchmark.Install)
			benchmarksToRun["overlap"].SubBenchmarks = overlapBenchmarksToRun
		}
		return benchmarksToRun
	}

	// If we get here, it means we need to execute everything installed
	benchmarksToRun = cfg.InstalledBenchmarks
	return benchmarksToRun
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	help := flag.Bool("h", false, "Help message")
	partition := flag.String("p", "", "Parition to use to submit the job (optional, relevant when a job manager such as Slurm is used)")
	device := flag.String("d", "", "Device to use (optional)")
	nActiveJobsFlag := flag.Int("max-running-jobs", 5, "The maximum of active running job at any given time (other jobs are queued and executed upon completion of running jobs)")
	ppnFlag := flag.Int("ppn", 1, "Number of MPI ranks per node (default: 1)")
	nNodesFlag := flag.Int("num-nodes", 1, "Number of nodes to use (default: 1)")
	longRunFlag := flag.Bool("long", false, "Run all supported tests, including tests not used to create the final metrics")
	osuUserSelectFlag := flag.Bool("osu", false, "Explicitly select OSU for execution. Only selected benchmarks will be executed")
	osuNonContigMemSelectFlag := flag.Bool("osu-noncontigmem", false, "Explicitly select OSU for non-contiguous memory for execution. Only selected benchmarks will be executed")
	smbSelectFlag := flag.Bool("smb", false, "Explicitly select SMB for execution. Only selected benchmarks will be executed")
	overlapSelectFlag := flag.Bool("overlap", false, "Explicitly select the overlap benchmark suite for execution. Only selected benchmarks will be executed")

	flag.Parse()

	if *help {
		filename := filepath.Base(os.Args[0])
		fmt.Printf("%s run openHPCA benchmarks\n", filename)
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	logFile := util.OpenLogFile("openhpca", "run")
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
	cfg.UserSelection.LongRun = *longRunFlag
	cfg.UserSelection.OsuSelected = *osuUserSelectFlag
	cfg.UserSelection.OsuNoncontigmemSelected = *osuNonContigMemSelectFlag
	cfg.UserSelection.SmbSelected = *smbSelectFlag
	cfg.UserSelection.OverlapSelected = *overlapSelectFlag

	// Load the configuration
	err := cfg.Load()
	if err != nil {
		fmt.Printf("Unable to load OpenHPCA configuration: %s\n", err)
		os.Exit(1)
	}

	/*
		jobmgr := jm.Detect()
		err = jm.Load(&jobmgr)
		if err != nil {
			fmt.Printf("Unable to load a job manager: %s\n", err)
		}
	*/

	cfg.DetectInstalledBenchmarks()

	// Some sanity checks
	if cfg.WP == nil {
		fmt.Println("ERROR: undefined workspace")
		os.Exit(1)
	}
	if !util.PathExists(cfg.WP.MpiDir) {
		fmt.Printf("ERROR: MPI installation directory '%s' is not valid\n", cfg.WP.MpiDir)
		os.Exit(1)
	}
	if *nActiveJobsFlag <= 0 {
		fmt.Printf("ERROR: the maximum number of active jobs mush be surperior to 0 (%d)\n", *nActiveJobsFlag)
		os.Exit(1)
	}

	r := experiments.NewRuntime()
	r.MaxRunningJobs = *nActiveJobsFlag
	r.ProgressFrequency = 5
	r.SleepBeforeSubmittingAgain = 1

	exps := new(experiments.Experiments)
	exps.NumResults = 1
	exps.MPICfg = new(experiments.MPIConfig)
	exps.MPICfg.MPI = new(implem.Info)
	exps.MPICfg.MPI.InstallDir = cfg.WP.MpiDir
	exps.Platform = new(platform.Info)
	exps.Platform.Name = *partition
	exps.Platform.Device = *device
	exps.Platform.MaxPPR = *ppnFlag
	exps.Platform.MaxNumNodes = *nNodesFlag
	exps.MaxExecTime = "1:00:00"

	benchmarksToRun := selectBenchmarksToRun(cfg)

	if *verbose {
		log.Printf("%d benchmarks being executed:\n", len(benchmarksToRun))
		for benchmarkName, installedBenchmark := range benchmarksToRun {
			for _, app := range installedBenchmark.SubBenchmarks {
				log.Printf(" - %s: %s\n", benchmarkName, app.Name)
			}
		}
	}

	for benchmarkName, installedBenchmark := range benchmarksToRun {
		for _, subBenchmark := range installedBenchmark.SubBenchmarks {
			e := new(experiments.Experiment)
			e.App = new(app.Info)
			e.App.Name = benchmarkName + "_" + subBenchmark.Name
			e.App.BinArgs = subBenchmark.BinArgs
			e.App.BinName = subBenchmark.BinName
			e.App.BinPath = subBenchmark.BinPath
			e.Name = e.App.Name
			if experimentIsStrictlyPointToPoint(e.Name) {
				e.Platform = new(platform.Info)
				e.Platform.Name = exps.Platform.Name
				e.Platform.Device = exps.Platform.Device
				e.Platform.MaxPPR = 1
				e.Platform.MaxNumNodes = 2
			}

			// Make sure to set special environment variables
			// todo: find a better way to abtract this, i.e., make sure it is set correctly for all MPI implementations
			overlapNumElts := os.Getenv("OPENHPCA_OVERLAP_MAX_NUM_ELTS")
			if overlapNumElts != "" && benchmarkName == "overlap" {
				e.MpirunArgs = append(e.MpirunArgs, "-x OPENHPCA_OVERLAP_MAX_NUM_ELTS="+overlapNumElts)
			}

			exps.List = append(exps.List, e)
		}
	}

	// Make sure the run directory exists and make sure it will be used when running experiments
	runDir := getRunDir(cfg)
	if !util.PathExists(runDir) {
		err = os.MkdirAll(runDir, 0777)
		if err != nil {
			fmt.Printf("ERROR: unable to create the run directory: %s", err)
			os.Exit(1)
		}
	}
	exps.RunDir = runDir
	exps.ResultsDir = runDir
	err = exps.Run(r)
	if err != nil {
		fmt.Printf("ERROR: unable to execute experiment: %s\n", err)
		os.Exit(1)
	}

	exps.Wait(r)
	r.Fini()
	log.Println("-> Job successfully executed")

	err = displayResults(cfg)
	if err != nil {
		fmt.Printf("ERROR: unable to display results: %s\n", err)
		fmt.Println("Some jobs executed by OpenHPCA may have failed because the default configuration needs to be customized for your configuration.")
		fmt.Printf("Please check the results in %s\n", cfg.GetRunDir())
		os.Exit(1)
	}
}
