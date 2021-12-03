//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package result

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/openucx/openhpca/tools/internal/pkg/analyser"
	"github.com/openucx/openhpca/tools/internal/pkg/overlap"
	"github.com/openucx/openhpca/tools/internal/pkg/score"
)

const (
	FilePermission = 0666
	FileName       = "openhpca_results.txt"
)

func String(dataDir string) (string, error) {
	metrics, err := analyser.ComputeScore(dataDir)
	if err != nil {
		return "", err
	}

	return metrics.ToString(), nil

}

func Create(outputDir string, dataDir string) error {
	filePath := filepath.Join(outputDir, FileName)
	content, err := String(dataDir)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, []byte(content), FilePermission)
	if err != nil {
		return err
	}
	return nil
}

func Save(path string, s *score.Metrics) error {
	content := s.ToString()
	return ioutil.WriteFile(path, []byte(content), FilePermission)
}

func ComputeOverlap(smbMPIOverhead float32, overlapData map[string][]string) (float32, map[string]float32, error) {
	numBenchs := len(overlapData)
	skipped := 0
	overlapDetails := make(map[string]float32)
	var finalOverlap float32
	finalOverlap = 0.0

	for benchName, output := range overlapData {
		if benchName == "overlap_ibarrier" {
			// We do not include data for ibarrier since it is difficult to have results for it
			// that makes statistical sense
			skipped++
			continue
		}
		overlapDetails[benchName] = 0.0
		for _, line := range output {
			if strings.HasPrefix(line, "Overlap: ") {
				line = strings.TrimPrefix(line, "Overlap: ")
				line = strings.TrimRight(line, "\n")
				line = strings.TrimSuffix(line, " %")
				value, err := strconv.ParseFloat(line, 32)
				if err != nil {
					return 0, nil, err
				}
				overlapDetails[benchName] = float32(value)
				finalOverlap += float32(value)
				break
			}
		}
	}
	if smbMPIOverhead < 0 {
		smbMPIOverhead = 0.0
	}
	finalOverlap += smbMPIOverhead
	overlapDetails["SMB mpi_overhead"] = smbMPIOverhead

	// Before we exit, we make sure we have all the expected data and if not, we add the
	// missing data as 0
	added := 0
	overlapBenchs := overlap.GetListSubBenchmarks()
	for _, subBench := range overlapBenchs {
		if _, ok := overlapDetails[subBench]; !ok {
			overlapDetails[subBench] = 0.0
			overlapData[subBench] = []string{"Data missing"}
			added++
		}
	}

	numBenchs -= skipped
	numBenchs += added
	return finalOverlap / float32(numBenchs+1), overlapDetails, nil
}
