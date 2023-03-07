//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package overlap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gvallee/go_benchmark/pkg/benchmark"
	"github.com/gvallee/go_hpc_jobmgr/pkg/implem"
	"github.com/gvallee/go_software_build/pkg/app"
	"github.com/gvallee/go_software_build/pkg/builder"
	fileutil "github.com/gvallee/go_util/pkg/util"
	"github.com/gvallee/go_workspace/pkg/workspace"
	"github.com/openucx/openhpca/tools/internal/pkg/util"
)

// Config represents the configuration of the overlap suite
type Config struct {
	URL string
}

// Install gathers all the data regarding the installation of the overlap suite so it can easily be looked up later on
type Install struct {
	Apps map[string]app.Info
}

const (
	overlapIallreduceID       = "overlap_iallreduce"
	overlapIallreduceBinName  = "overlap_iallreduce"
	overlapIreduceID          = "overlap_ireduce"
	overlapIreduceBinName     = "overlap_ireduce"
	overlapIallgatherID       = "overlap_iallgather"
	overlapIallgatherBinName  = "overlap_iallgather"
	overlapIallgathervID      = "overlap_iallgatherv"
	overlapIallgathervBinName = "overlap_iallgatherv"
	overlapIalltoallID        = "overlap_ialltoall"
	overlapIalltoallBinName   = "overlap_ialltoall"
	overlapIalltoallvID       = "overlap_ialltoallv"
	overlapIalltoallvBinName  = "overlap_ialltoallv"
	overlapIbarrierID         = "overlap_ibarrier"
	overlapIbarrierBinName    = "overlap_ibarrier"
	overlapIbcastID           = "overlap_ibcast"
	overlapIbcastBinName      = "overlap_ibcast"
	overlapIgatherID          = "overlap_igather"
	overlapIgatherBinName     = "overlap_igather"
	overlapIgathervID         = "overlap_igatherv"
	overlapIgathervBinName    = "overlap_igatherv"
)

var RequiredBenchmarks = []string{overlapIallreduceID, overlapIreduceID, overlapIallgatherID, overlapIallgathervID,
	overlapIalltoallID, overlapIalltoallvID, overlapIbcastID, overlapIgatherID, overlapIgathervID}

// ParseCfg is the function to invoke to parse lines from the main configuration files
// that are specific to the overlap suite
func ParseCfg(cfg *benchmark.Config, basedir string, srcDir string, key string, value string) {
	switch key {
	case "URL":
		// Replace OPENHPCA_DIR by the actual value of where the OpenHPCA code sits
		// since the overlap suite is shipped with OpenHPCA
		cfg.URL = util.UpdateOpenHPCADirValue(value, basedir)
	}
}

func GetListSubBenchmarks() []string {
	return []string{overlapIallreduceID, overlapIreduceID, overlapIallgatherID, overlapIallgathervID, overlapIalltoallID, overlapIalltoallvID, overlapIbarrierID, overlapIbcastID, overlapIgatherID, overlapIgathervID}
}

func GetSubBenchmarks(cfg *benchmark.Config, wp *workspace.Config) map[string]app.Info {
	m := make(map[string]app.Info)

	overlapDir := strings.TrimPrefix(cfg.URL, "file://")
	installDir := filepath.Join(wp.InstallDir, "overlap")

	overlapIallgatherInfo := app.Info{
		Name: overlapIallgatherID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIallgatherID),
		},
		BinName: overlapIallgatherBinName,
		BinPath: filepath.Join(installDir, "overlap", overlapIallgatherID),
		BinArgs: nil,
	}
	m[overlapIallgatherID] = overlapIallgatherInfo

	overlapIallgathervInfo := app.Info{
		Name: overlapIallgathervID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIallgathervID),
		},
		BinName: overlapIallgathervBinName,
		BinPath: filepath.Join(installDir, "overlap", overlapIallgathervID),
		BinArgs: nil,
	}
	m[overlapIallgathervID] = overlapIallgathervInfo

	overlapIallreduceInfo := app.Info{
		Name: overlapIallreduceID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIallreduceID),
		},
		BinName: overlapIallreduceBinName,
		BinPath: filepath.Join(installDir, "overlap", overlapIallreduceID),
		BinArgs: nil,
	}
	m[overlapIallreduceID] = overlapIallreduceInfo

	overlapIreduceInfo := app.Info{
		Name: overlapIreduceID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIreduceID),
		},
		BinName: overlapIreduceBinName,
		BinPath: filepath.Join(installDir, "overlap", overlapIreduceID),
		BinArgs: nil,
	}
	m[overlapIreduceID] = overlapIreduceInfo

	overlapIalltoallInfo := app.Info{
		Name: overlapIalltoallID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIalltoallID),
		},
		BinName: overlapIalltoallBinName,
		BinPath: filepath.Join(installDir, "overlap", overlapIalltoallID),
		BinArgs: nil,
	}
	m[overlapIalltoallID] = overlapIalltoallInfo

	overlapIalltoallvInfo := app.Info{
		Name: overlapIalltoallvID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIalltoallvID),
		},
		BinName: overlapIalltoallvID,
		BinPath: filepath.Join(installDir, "overlap", overlapIalltoallvID),
		BinArgs: nil,
	}
	m[overlapIalltoallvID] = overlapIalltoallvInfo

	overlapIbarrierInfo := app.Info{
		Name: overlapIbarrierID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIbarrierID),
		},
		BinName: overlapIbarrierID,
		BinPath: filepath.Join(installDir, "overlap", overlapIbarrierID),
		BinArgs: nil,
	}
	m[overlapIbarrierID] = overlapIbarrierInfo

	overlapIbcast := app.Info{
		Name: overlapIbcastID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIbcastID),
		},
		BinName: overlapIbcastID,
		BinPath: filepath.Join(installDir, "overlap", overlapIbcastID),
		BinArgs: nil,
	}
	m[overlapIbcastID] = overlapIbcast

	overlapIgatherInfo := app.Info{
		Name: overlapIgatherID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIgatherID),
		},
		BinName: overlapIgatherBinName,
		BinPath: filepath.Join(installDir, "overlap", overlapIgatherID),
		BinArgs: nil,
	}
	m[overlapIgatherID] = overlapIgatherInfo

	overlapIgathervInfo := app.Info{
		Name: overlapIgathervID,
		Source: app.SourceCode{
			URL: "file:///" + filepath.Join(overlapDir, overlapDir, overlapIgathervID),
		},
		BinName: overlapIgathervBinName,
		BinPath: filepath.Join(installDir, "overlap", overlapIgathervID),
		BinArgs: nil,
	}
	m[overlapIgathervID] = overlapIgathervInfo

	return m
}

// Compile downloads and installs the overlap suite on the host
func Compile(cfg *benchmark.Config, wp *workspace.Config) (*benchmark.Install, error) {
	// Find MPI and make sure we pass the information about it to the builder
	mpiInfo := new(implem.Info)
	mpiInfo.InstallDir = wp.MpiDir
	err := mpiInfo.Load(nil)
	if err != nil {
		return nil, fmt.Errorf("no suitable MPI available: %s", err)
	}

	pathEnv := os.Getenv("PATH")
	pathEnv = "PATH=" + filepath.Join(mpiInfo.InstallDir, "bin") + ":" + pathEnv
	ldPathEnv := os.Getenv("LD_LIBRARY_PATH")
	ldPathEnv = "LD_LIBRARY_PATH=" + filepath.Join(mpiInfo.InstallDir, "lib") + ":" + ldPathEnv

	b := new(builder.Builder)
	b.Persistent = wp.InstallDir
	b.App.Name = "overlap"
	b.App.Source.URL = cfg.URL

	if wp.ScratchDir == "" || wp.InstallDir == "" || wp.BuildDir == "" || wp.SrcDir == "" {
		return nil, fmt.Errorf("invalid workspace")
	}
	b.Env.ScratchDir = wp.ScratchDir
	b.Env.InstallDir = wp.InstallDir
	b.Env.BuildDir = wp.BuildDir
	b.Env.SrcDir = filepath.Join(wp.SrcDir, b.App.Name)
	b.Env.Env = append(b.Env.Env, pathEnv)
	b.Env.Env = append(b.Env.Env, ldPathEnv)
	ccEnv := "CC=" + filepath.Join(mpiInfo.InstallDir, "bin", "mpicc")
	b.Env.Env = append(b.Env.Env, ccEnv)
	cxxEnv := "CXX=" + filepath.Join(mpiInfo.InstallDir, "bin", "mpicxx")
	b.Env.MakeExtraArgs = append(b.Env.MakeExtraArgs, ccEnv)
	b.Env.MakeExtraArgs = append(b.Env.MakeExtraArgs, cxxEnv)

	// fixme: ultimately, we want a persistent install and instead of passing in
	// 'true' to say so, we want to pass in the install directory
	err = b.Load(false)
	if err != nil {
		return nil, err
	}

	// Finally we can install the overlap benchmark, it is all ready
	res := b.Install()
	if res.Err != nil {
		return nil, res.Err
	}

	installInfo := new(benchmark.Install)
	installInfo.SubBenchmarks = append(installInfo.SubBenchmarks, b.App)

	return installInfo, nil
}

// DetectInstall scans a specific workspace and detect which overlap benchmarks are
// available. It sets the path to the binary during the detection so it is easier
// to start benchmarks later on.
func DetectInstall(cfg *benchmark.Config, wp *workspace.Config) *benchmark.Install {
	log.Println("Detecting overlap installation...")
	installInfo := new(benchmark.Install)
	overlapBenchmarks := GetSubBenchmarks(cfg, wp)
	for benchmarkName, benchmarkInfo := range overlapBenchmarks {
		log.Printf("-> Checking if %s is installed...", benchmarkName)
		if fileutil.FileExists(benchmarkInfo.BinPath) {
			log.Printf("\t%s exists", benchmarkInfo.BinPath)
			installInfo.SubBenchmarks = append(installInfo.SubBenchmarks, benchmarkInfo)
		} else {
			log.Printf("\t%s does not exist", benchmarkInfo.BinPath)
		}
	}
	return installInfo
}

// Display shows the current overlap benchmark suite configuration
func Display(cfg *benchmark.Config) {
	fmt.Printf("\toverlap benchmark suite URL: %s\n", cfg.URL)
}
