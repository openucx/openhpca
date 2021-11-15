//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gvallee/go_benchmark/pkg/benchmark"
	"github.com/gvallee/go_osu/pkg/osu"
	"github.com/gvallee/go_util/pkg/util"
	"github.com/gvallee/go_workspace/pkg/workspace"
	"github.com/gvallee/openhpca/tools/internal/pkg/download"
	"github.com/gvallee/openhpca/tools/internal/pkg/overlap"
	"github.com/gvallee/openhpca/tools/internal/pkg/smb"
	openhpcautil "github.com/gvallee/openhpca/tools/internal/pkg/util"
)

const (
	// OpenHPCAConfigFilename is the name of the tool's configuration file
	OpenHPCAConfigFilename = "openhpca.conf"

	// OpenHPCABaseDir is the OpenHPCA's configuration base directory on the host
	OpenHPCABaseDir = ".openhpca"

	// OpenHPCAWorkspaceConfigFilename is the filename of a workspace configuration file
	OpenHPCAWorkspaceConfigFilename = "workspace.conf"

	// OpenHPCASrcDir is the default name of the directory where software sources are saved
	OpenHPCASrcDir = "src"
)

// SlurmCfg represents the Slurm configuration requested by the user through a configuration file
type SlurmCfg struct {
	Partition string
}

// AppsCfg represents the configuration of all the applications/benchmarks used by OpenHPCA
type AppsCfg struct {
	// OSUCfg represents the configuration of the OSU micro-benchmarks
	OSUCfg benchmark.Config

	// OSUNonContigMem represents the configuration of the modified OSU micro-benchmarks for non-contiguous memory
	OSUNonContigMem benchmark.Config

	// SMBCfg represents the configuration of the SMB benchmarks
	SMBCfg benchmark.Config

	// OverlapCfg represents the configuration of the overlap benchmark suite
	OverlapCfg benchmark.Config
}

// Data represents the configuration of the file, mainly based
// on what is in the configuration file
type Data struct {
	// Basedir is the base directory where the code is
	Basedir string

	// BinName is the name of the setup binary
	BinName string

	// ConfigFile is the path the private OpenHPCA configuration file
	ConfigFile string

	// WP is the configuration of the current workspace
	WP *workspace.Config

	// Apps is the configuration of all the benchmarks used by OpenHPCA
	Apps AppsCfg

	// InstalledBenchmarks is the list of benchmarks that are available for execution
	InstalledBenchmarks map[string]*benchmark.Install

	// fixme: should be moved to go_software_build
	Downloader *download.Downloader

	// Slurm represents the configuration of Slurm, including users' parameters (e.g., partition)
	Slurm SlurmCfg
}

func cleanupLine(line string) string {
	line = strings.TrimLeft(line, " ")
	line = strings.TrimLeft(line, "\t")
	line = strings.TrimRight(line, " ")
	line = strings.TrimRight(line, "\t")
	return line
}

func (cfg *Data) analyzeWorkspaceCfgKeyValue(blockName string, key string, value string) error {
	switch blockName {
	case "":
		switch key {
		case "dir":
			cfg.WP.Basedir = value
		case "MPI":
			fallthrough
		case "mpi":
			cfg.WP.MpiDir = value
		case "mpirun_args":
			cfg.WP.MpirunArgs = value
		}
	case "Slurm":
		fallthrough
	case "SLURM":
		fallthrough
	case "slurm":
		switch key {
		case "partition":
			cfg.Slurm.Partition = value
		}
	default:
		return fmt.Errorf("analyzeWorkspaceCfgKeyValue(): unknown block name: %s", blockName)
	}

	return nil
}

func (cfg *Data) analyzeToolCfgKeyValue(blockName string, key string, value string) error {
	switch blockName {
	case "osu_noncontig_mem":
		osu.ParseCfg(&cfg.Apps.OSUNonContigMem, cfg.Basedir, openhpcautil.OpenHPCADirID, cfg.WP.SrcDir, key, value)
	case "OSU":
		osu.ParseCfg(&cfg.Apps.OSUCfg, cfg.Basedir, openhpcautil.OpenHPCADirID, cfg.WP.SrcDir, key, value)
	case "SMB":
		smb.ParseCfg(&cfg.Apps.SMBCfg, cfg.Basedir, cfg.WP.SrcDir, key, value)
	case "overlap":
		overlap.ParseCfg(&cfg.Apps.OverlapCfg, cfg.Basedir, cfg.WP.SrcDir, key, value)
	default:
		return fmt.Errorf("analyzeToolCfgKeyValue(): unknown block name: %s", blockName)
	}

	return nil
}

func (cfg *Data) parseConfig(context string, content string) error {
	lines := strings.Split(content, "\n")
	blockName := ""
	for _, line := range lines {
		line = cleanupLine(line)
		if line == "" {
			// Skip empty lines
			continue
		}
		if strings.HasPrefix(line, "#") {
			// Comment, skip the line
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// New block
			blockName = strings.TrimPrefix(line, "[")
			blockName = strings.TrimSuffix(blockName, "]")
		}

		if strings.Contains(line, "=") {
			tokens := strings.Split(line, "=")
			if len(tokens) != 2 {
				return fmt.Errorf("invalid format: %s", line)
			}
			key := cleanupLine(tokens[0])
			value := cleanupLine(tokens[1])
			switch context {
			case "type":
				err := cfg.analyzeToolCfgKeyValue(blockName, key, value)
				if err != nil {
					return err
				}
			case "tool":
				err := cfg.analyzeToolCfgKeyValue(blockName, key, value)
				if err != nil {
					return err
				}
			case "wp":
				err := cfg.analyzeWorkspaceCfgKeyValue(blockName, key, value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (cfg *Data) check() error {
	if cfg.Basedir == "" || !util.PathExists(cfg.Basedir) {
		return fmt.Errorf("basedir %s does not exit", cfg.Basedir)
	}

	// Make sure ~/.openhpca exists, if not create it
	if os.Getenv("HOME") == "" {
		fmt.Println("HOME environment variable not defined")
		os.Exit(1)
	}
	hpcaBasedir := filepath.Join(os.Getenv("HOME"), OpenHPCABaseDir)
	if !util.PathExists(hpcaBasedir) {
		err := os.Mkdir(hpcaBasedir, 0700)
		if err != nil {
			fmt.Printf("Unable to create %s\n", hpcaBasedir)
			os.Exit(1)
		}
	}

	return nil
}

// Load reads the configuration files and loads the configuration in a structure
func (cfg *Data) Load() error {
	err := cfg.check()
	if err != nil {
		return err
	}

	cfg.WP = new(workspace.Config)
	cfg.ConfigFile = path.Join(cfg.Basedir, OpenHPCAConfigFilename)
	hpcaBasedir := filepath.Join(os.Getenv("HOME"), OpenHPCABaseDir)
	cfg.WP.ConfigFile = filepath.Join(hpcaBasedir, OpenHPCAWorkspaceConfigFilename)

	// Load the tool's configuration file
	log.Println("-> Parsing the OpenHPCA configuration file...")
	content, err := ioutil.ReadFile(cfg.ConfigFile)
	if err != nil {
		return err
	}
	contentStr := string(content)
	err = cfg.parseConfig("tool", contentStr)
	if cfg == nil || err != nil {
		return err
	}

	// Load the workspace configuration file if the file exists and make sure the workspace is ready to go
	log.Printf("-> Parsing the workspace configuration file %s", cfg.WP.ConfigFile)
	if util.FileExists(cfg.WP.ConfigFile) {
		content, err = ioutil.ReadFile(cfg.WP.ConfigFile)
		if err != nil {
			return err
		}
		contentStr = string(content)
		err = cfg.parseConfig("wp", contentStr)
		if cfg == nil || err != nil {
			return err
		}
	} else {
		err = fmt.Errorf("no workspace has been defined, please run '%s -init-workspace' to create a default workspace", cfg.BinName)
		log.Printf("%s", err)
		return err
	}
	err = cfg.WP.Init()
	if err != nil {
		return err
	}

	return nil
}

// InitWorkspace creates a default workspace
func (cfg *Data) InitWorkspace() error {
	err := cfg.check()
	if err != nil {
		return err
	}

	if util.FileExists(cfg.WP.ConfigFile) {
		return fmt.Errorf("a workspace is already defined: %s; please delete to initialize a new workspace", cfg.WP.ConfigFile)
	}

	// Create a default workspace configuration
	content := "dir=" + filepath.Join(os.Getenv("HOME"), OpenHPCABaseDir, "wp") + "\n"
	f, err := os.Create(cfg.WP.ConfigFile)
	if err != nil {
		return err
	}
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

// DetectInstalledBenchmarks goes through all the supported benchmarks and
// detects which once are available for execution
func (cfg *Data) DetectInstalledBenchmarks() {
	cfg.InstalledBenchmarks = make(map[string]*benchmark.Install)

	osuInstalledOSUFlavors := osu.DetectInstall(&cfg.Apps.OSUCfg, cfg.WP)
	if osuInstalledOSUFlavors != nil {
		if _, ok := osuInstalledOSUFlavors[osu.OSUBaseDir]; ok {
			cfg.InstalledBenchmarks["osu"] = osuInstalledOSUFlavors[osu.OSUBaseDir]
		}

		if _, ok := osuInstalledOSUFlavors[osu.OSUNonConfigMemBaseDir]; ok {
			cfg.InstalledBenchmarks["osu_noncontig_mem"] = osuInstalledOSUFlavors[osu.OSUNonConfigMemBaseDir]
		}
	}

	smbInstalledBenchmarks := smb.DetectInstall(&cfg.Apps.SMBCfg, cfg.WP)
	cfg.InstalledBenchmarks["smb"] = smbInstalledBenchmarks

	overlapInstalledBenchmarks := overlap.DetectInstall(&cfg.Apps.OverlapCfg, cfg.WP)
	cfg.InstalledBenchmarks["overlap"] = overlapInstalledBenchmarks
}

// Display shows the current configuration
func (cfg *Data) Display() {
	fmt.Println("OpenHPCA configuration:")
	fmt.Printf("\tConfiguration file: %s\n", cfg.ConfigFile)
	osu.Display(&cfg.Apps.OSUCfg)
	osu.Display(&cfg.Apps.OSUNonContigMem)
	smb.Display(&cfg.Apps.SMBCfg)
	overlap.Display(&cfg.Apps.OverlapCfg)

	if cfg.WP.Basedir != "" {
		fmt.Println("\nWorkspace configuration:")
		fmt.Printf("\tConfiguration file: %s\n", cfg.WP.ConfigFile)
		fmt.Printf("\tWorkspace directory: %s\n", cfg.WP.Basedir)
	} else {
		fmt.Println("\nNo custum workspace has been defined")
	}

	fmt.Println("Installed benchmarks:")
	for name, installedBenchmark := range cfg.InstalledBenchmarks {
		fmt.Printf("\t-> %s\n", name)
		for _, benchmarkInfo := range installedBenchmark.SubBenchmarks {
			fmt.Printf("\t\t%s: %s\n", benchmarkInfo.Name, benchmarkInfo.BinPath)
		}
	}
}

func (cfg *Data) GetRunDir() string {
	return filepath.Join(cfg.WP.Basedir, "run")
}
