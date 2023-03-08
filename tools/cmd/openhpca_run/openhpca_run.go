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
	"strconv"
	"time"

	"github.com/gvallee/go_benchmark/pkg/benchmark"
	"github.com/gvallee/go_hpc_jobmgr/pkg/implem"
	"github.com/gvallee/go_hpc_jobmgr/pkg/mpi"
	"github.com/gvallee/go_software_build/pkg/app"
	"github.com/gvallee/go_util/pkg/util"
	"github.com/gvallee/validation_tool/pkg/experiments"
	"github.com/gvallee/validation_tool/pkg/platform"
	"github.com/openucx/openhpca/tools/internal/pkg/config"
	"github.com/openucx/openhpca/tools/internal/pkg/fileUtils"
	"github.com/openucx/openhpca/tools/internal/pkg/overlap"
	"github.com/openucx/openhpca/tools/internal/pkg/report"
	"github.com/openucx/openhpca/tools/internal/pkg/score"
	"github.com/openucx/openhpca/tools/internal/pkg/smb"
)

func displayResults(cfg *config.Data) error {
	runDir := cfg.GetRunDir()
	m, err := score.Compute(runDir)
	if err != nil {
		return fmt.Errorf("unable to compute the metrics: %w", err)
	}
	resultsStr := m.ToString()
	fmt.Printf("\nOpenHPCA:\n" + resultsStr)
	resultFile := filepath.Join(cfg.Basedir, "..", score.FileName)
	err = ioutil.WriteFile(resultFile, []byte(resultsStr), fileUtils.DefaultPermission)
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
	if cfg.UserParams.BenchSelection.OsuNoncontigmemSelected &&
		cfg.UserParams.BenchSelection.OsuSelected &&
		cfg.UserParams.BenchSelection.SmbSelected &&
		cfg.UserParams.BenchSelection.OverlapSelected {
		return true
	}
	return false
}

func userSelectedAtLeastOneBenchmark(cfg *config.Data) bool {
	if cfg.UserParams.BenchSelection.OsuNoncontigmemSelected ||
		cfg.UserParams.BenchSelection.OsuSelected ||
		cfg.UserParams.BenchSelection.SmbSelected ||
		cfg.UserParams.BenchSelection.OverlapSelected {
		return true
	}
	return false
}

func selectBenchmarksToRun(cfg *config.Data) map[string]*benchmark.Install {
	var benchmarksToRun map[string]*benchmark.Install
	benchmarksToRun = make(map[string]*benchmark.Install)

	// Check the options to make sure we know what is required
	if userSelectedAllBenchmarks(cfg) {
		cfg.UserParams.BenchSelection.LongRun = true
	}

	if !userSelectedAtLeastOneBenchmark(cfg) && !cfg.UserParams.BenchSelection.LongRun {
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

	if !cfg.UserParams.BenchSelection.LongRun && userSelectedAtLeastOneBenchmark(cfg) {
		// Get the list of selected benchmarks by the user
		if cfg.UserParams.BenchSelection.OsuSelected {
			var osuBenchmarksToRun []app.Info
			installedOSUSubBenchmarks := cfg.InstalledBenchmarks["osu"]
			osuBenchmarksToRun = append(osuBenchmarksToRun, installedOSUSubBenchmarks.SubBenchmarks...)
			benchmarksToRun["osu"] = new(benchmark.Install)
			benchmarksToRun["osu"].SubBenchmarks = osuBenchmarksToRun
		}

		if cfg.UserParams.BenchSelection.OsuNoncontigmemSelected {
			var osuBenchmarksToRun []app.Info
			installedOSUNoncontigmemSubBenchmarks := cfg.InstalledBenchmarks["osu_noncontig_mem"]
			osuBenchmarksToRun = append(osuBenchmarksToRun, installedOSUNoncontigmemSubBenchmarks.SubBenchmarks...)
			benchmarksToRun["osu"] = new(benchmark.Install)
			benchmarksToRun["osu"].SubBenchmarks = osuBenchmarksToRun
		}

		if cfg.UserParams.BenchSelection.SmbSelected {
			var smbBenchmarksToRun []app.Info
			installedSMBSubBenchmarks := cfg.InstalledBenchmarks["smb"]
			smbBenchmarksToRun = append(smbBenchmarksToRun, installedSMBSubBenchmarks.SubBenchmarks...)
			benchmarksToRun["smb"] = new(benchmark.Install)
			benchmarksToRun["smb"].SubBenchmarks = smbBenchmarksToRun
		}

		if cfg.UserParams.BenchSelection.OverlapSelected {
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
	nNodesFlag := flag.Int("num-nodes", 2, "Number of nodes to use (default: 2)")
	longRunFlag := flag.Bool("long", false, "Run all supported tests, including tests not used to create the final metrics")
	osuUserSelectFlag := flag.Bool("osu", false, "Explicitly select OSU for execution. Only selected benchmarks will be executed")
	osuNonContigMemSelectFlag := flag.Bool("osu-noncontigmem", false, "Explicitly select OSU for non-contiguous memory for execution. Only selected benchmarks will be executed")
	smbSelectFlag := flag.Bool("smb", false, "Explicitly select SMB for execution. Only selected benchmarks will be executed")
	overlapSelectFlag := flag.Bool("overlap", false, "Explicitly select the overlap benchmark suite for execution. Only selected benchmarks will be executed")
	overlapConfigFilePathFlag := flag.String("overlap-config", "", "Path to the overlap configuration file. An example is available there: 'etc/examples/overlap_conf.json'")

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
	cfg.UserParams.Set = true
	cfg.UserParams.StartTime = report.CreateTimestampString(time.Now())
	cfg.UserParams.Device = *device
	cfg.UserParams.NumActiveJobs = *nActiveJobsFlag
	cfg.UserParams.NumNodes = *nNodesFlag
	cfg.UserParams.PPN = *ppnFlag
	cfg.UserParams.Partition = *partition
	cfg.UserParams.BenchSelection.LongRun = *longRunFlag
	cfg.UserParams.BenchSelection.OsuSelected = *osuUserSelectFlag
	cfg.UserParams.BenchSelection.OsuNoncontigmemSelected = *osuNonContigMemSelectFlag
	cfg.UserParams.BenchSelection.SmbSelected = *smbSelectFlag
	cfg.UserParams.BenchSelection.OverlapSelected = *overlapSelectFlag

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

	overlapConfig := new(overlap.Config)
	if *overlapConfigFilePathFlag != "" {
		err := overlapConfig.LoadConfig(*overlapConfigFilePathFlag)
		if err != nil {
			fmt.Printf("ERROR: unable to load overlap configuration: %s\n", err)
			os.Exit(1)
		}
	}

	// Detect the MPI implementation so we can properly customize the environment
	localMPI, err := mpi.DetectFromDir(cfg.WP.MpiDir)
	if err != nil {
		fmt.Printf("unable to detect the MPI implementation installed in %s: %s\n", cfg.WP.MpiDir, err)
		os.Exit(1)
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

			//For SMB msgrate tests, add ppn, peers to BinArgs
			if e.Name == "smb_msgrate" || e.Name == "smb_rma_mt_mpi" {
				e.App.BinArgs = append(e.App.BinArgs,
					fmt.Sprintf("-p %d -n %d",
						exps.Platform.MaxNumNodes-1,
						exps.Platform.MaxPPR))
			}

			// Make sure to set special environment variables
			// todo: find a better way to abtract this, i.e., make sure it is set correctly for all MPI implementations
			// Data from the overlap configuration file always prevail on the environment variable from the calling
			// process
			overlapNumElts := os.Getenv(overlap.MaxNumEltsEnvVar)
			if overlapConfig.MaxNumEltsLookupTable != nil {
				overlapNumElts = strconv.Itoa(overlapConfig.MaxNumEltsLookupTable[subBenchmark.BinName])
			}
			if overlapNumElts != "" && benchmarkName == "overlap" {
				if localMPI.ID == implem.OMPI {
					e.MpirunArgs = append(e.MpirunArgs, "-x "+overlap.MaxNumEltsEnvVar+"="+overlapNumElts)
				}
				if localMPI.ID == implem.MPICH || localMPI.ID == implem.MVAPICH2 {
					e.MpirunArgs = append(e.MpirunArgs, "-genv "+overlap.MaxNumEltsEnvVar+"="+overlapNumElts)
				}
			}

			exps.List = append(exps.List, e)
		}
	}

	// Make sure the run directory exists and make sure it will be used when running experiments
	runDir := cfg.GetRunDir()
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

		// Generate the report even if case of errors to make it easier to identify
		// what are the errors
		err = report.Generate(cfg)
		if err != nil {
			fmt.Printf("ERROR: unable to generate the report: %s\n", err)
		}
		os.Exit(1)
	}

	err = report.Generate(cfg)
	if err != nil {
		fmt.Printf("ERROR: unable to generate the report: %s\n", err)
		os.Exit(1)
	}
}
