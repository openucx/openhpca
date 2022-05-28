//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package smb

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

const (
	mpiOverheadID      = "mpi_overhead"
	mpiOverheadBinName = "mpi_overhead"
	rmaMTmpiID         = "rma_mt_mpi"
	rmaMTmpiBinName    = "msgrate"
	msgrateID          = "msgrate"
	msgrateBinName     = "msgrate"

	OverheadResultFilePrefix = "smb_mpi_overhead"

	defaultMsgSize = 8
)

var RequiredBenchmarks = []string{mpiOverheadID}

// Config represents the configuration of SMB
type Config struct {
	URL string
}

// Install gathers all the data regarding the installation of SMB so it can easily be looked up later on
type Install struct {
	Apps map[string]app.Info
}

// ParseCfg is the function to invoke to parse lines from the main configuration files
// that are specific to SMB
func ParseCfg(cfg *benchmark.Config, basedir string, srcDir string, key string, value string) {
	switch key {
	case "URL":
		// Replace OPENHPCA_DIR by the actual value of where the OpenHPCA code sits
		// since SMB is a submodule
		cfg.URL = util.UpdateOpenHPCADirValue(value, basedir)
	}
}

func getSubBenchmarks(cfg *benchmark.Config, wp *workspace.Config) map[string]app.Info {
	m := make(map[string]app.Info)

	smbDir := strings.TrimPrefix(cfg.URL, "file://")

	mpiOverheadInfo := app.Info{
		Name:    mpiOverheadID,
		URL:     "file:///" + filepath.Join(smbDir, "src", mpiOverheadID),
		BinName: mpiOverheadBinName,
		BinPath: filepath.Join(wp.InstallDir, mpiOverheadID, mpiOverheadID, mpiOverheadBinName),
		BinArgs: []string{" --msgsize", " ", fmt.Sprintf("%d", defaultMsgSize)},
	}
	m[mpiOverheadID] = mpiOverheadInfo

	msgrateInfo := app.Info{
		Name: "msgrate",
		URL:  "file:///" + filepath.Join(smbDir, "src", msgrateID),
		BinName: msgrateBinName,
		BinPath: filepath.Join(wp.InstallDir, msgrateID, msgrateID, msgrateBinName),
	}
	m["msgrate"] = msgrateInfo

	rmaMtMpiInfo := app.Info{
		Name:    rmaMTmpiID,
		URL:     "file:///" + filepath.Join(smbDir, "src", rmaMTmpiID),
		BinName: rmaMTmpiBinName,
		BinPath: filepath.Join(wp.InstallDir, rmaMTmpiID, rmaMTmpiID, rmaMTmpiBinName),
	}
	m[rmaMTmpiID] = rmaMtMpiInfo

	/*
		this sub-benchmark seems to create compile errors
		shmemMtInfo := app.Info{
			Name: "shmem_mt",
			URL:  "file:///" + filepath.Join(smbDir, "src", "shmem_mt"),
		}
		m["shmem_mt"] = shmemMtInfo
	*/

	return m
}

// Compile downloads and installs SMB on the host
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

	benchmarks := getSubBenchmarks(cfg, wp)
	installInfo := new(benchmark.Install)
	for _, info := range benchmarks {
		b := new(builder.Builder)
		b.Persistent = wp.InstallDir
		b.App = info

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
		err := b.Load(false)
		if err != nil {
			return nil, err
		}

		// Finally we can install SMB, it is all ready
		res := b.Install()
		if res.Err != nil {
			return nil, res.Err
		}

		installInfo.SubBenchmarks = append(installInfo.SubBenchmarks, info)
	}

	return installInfo, nil
}

// DetectInstall scans a specific workspace and detect which SMB benchmarks are
// available. It sets the path to the binary during the detection so it is easier
// to start benchmarks later on.
func DetectInstall(cfg *benchmark.Config, wp *workspace.Config) *benchmark.Install {
	log.Println("Detecting SMB installation...")
	installInfo := new(benchmark.Install)
	smbBenchmarks := getSubBenchmarks(cfg, wp)
	for benchmarkName, benchmarkInfo := range smbBenchmarks {
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

// Display shows the current SMB configuration
func Display(cfg *benchmark.Config) {
	fmt.Printf("\tSMB URL: %s\n", cfg.URL)
}
