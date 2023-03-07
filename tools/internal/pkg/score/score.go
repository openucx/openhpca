//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package score

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/openucx/openhpca/tools/internal/pkg/fileUtils"
	"github.com/openucx/openhpca/tools/internal/pkg/result"
)

const (
	FileName = "score.txt"
)

// Metrics gathers all the data that represents the final result of the benchmark suite
type Metrics struct {
	Bandwidth      float64
	BandwidthUnit  string
	Latency        float64
	LatencyUnit    string
	MpiOverlap     float32
	Score          int
	OverlapData    map[string][]string
	OverlapScore   float32
	OverlapDetails map[string]float32
}

func Compute(dataDir string) (*Metrics, error) {
	data, err := result.Get(dataDir)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, fmt.Errorf("unable to load results")
	}

	metrics := new(Metrics)
	metrics.Bandwidth = data.Bandwidth
	metrics.BandwidthUnit = data.BandwidthUnit
	metrics.Latency = float64(data.Latency)
	metrics.LatencyUnit = data.LatencyUnit
	metrics.OverlapScore = data.MpiOverhead

	if data.BandwidthUnit != "Gb/s" {
		return nil, fmt.Errorf("unsupported unit for bandwidth (%s)", data.BandwidthUnit)
	}
	if data.LatencyUnit != "us" {
		return nil, fmt.Errorf("unsupported unit for latency (%s)", data.LatencyUnit)
	}

	return metrics, nil
}

func (s *Metrics) ToString() string {
	content := fmt.Sprintf("Bandwidth: %f %s\n", s.Bandwidth, s.BandwidthUnit)
	content += fmt.Sprintf("Latency: %f %s\n", s.Latency, s.LatencyUnit)
	content += fmt.Sprintf("Overlap: %f\n", s.OverlapScore)
	content += fmt.Sprintf("\t- MPI overlap: %f %%\n", s.MpiOverlap)
	for benchmarkName, results := range s.OverlapData {
		content += "\t- " + benchmarkName + ":\n"
		for _, line := range results {
			content += "\t\t" + line + "\n"
		}
		content += "\n"
	}
	content += "\n"
	//content += fmt.Sprintf("Score: %d\n", s.Score)
	return content
}

func (s *Metrics) Save(path string) error {
	content := s.ToString()
	return ioutil.WriteFile(path, []byte(content), fileUtils.DefaultPermission)
}

func Create(outputDir string, dataDir string) error {
	filePath := filepath.Join(outputDir, FileName)
	m, err := Compute(dataDir)
	if err != nil {
		return fmt.Errorf("unable to compute the metrics: %w", err)
	}
	err = m.Save(filePath)
	if err != nil {
		return err
	}
	return nil
}
