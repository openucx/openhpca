//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package config

import (
	"fmt"

	"github.com/gvallee/go_benchmark/pkg/benchmark"
	"github.com/gvallee/go_osu/pkg/osu"
	"github.com/gvallee/openhpca/tools/internal/pkg/overlap"
	"github.com/gvallee/openhpca/tools/internal/pkg/smb"
)

// Compile makes sure all the required software is properly installed and compiled
func (cfg *Data) Compile() error {
	cfg.InstalledBenchmarks = make(map[string]*benchmark.Install)

	// Compile OSU
	_, err := osu.Compile(&cfg.Apps.OSUCfg, cfg.WP, osu.OSUBaseDir)
	if err != nil {
		return fmt.Errorf("unable to compile OSU: %w", err)
	}

	// Compile osu_noncontig_mem
	_, err = osu.Compile(&cfg.Apps.OSUNonContigMem, cfg.WP, osu.OSUNonConfigMemBaseDir)
	if err != nil {
		return fmt.Errorf("unable to compile OSU for non-contiguous memory: %w", err)
	}

	// Compile SMB
	_, err = smb.Compile(&cfg.Apps.SMBCfg, cfg.WP)
	if err != nil {
		return fmt.Errorf("unable to compile SMB: %w", err)
	}

	// Compile the overlap benchmark suite
	_, err = overlap.Compile(&cfg.Apps.OverlapCfg, cfg.WP)
	if err != nil {
		return fmt.Errorf("unable to compile the overlap: %w", err)
	}

	return nil
}
