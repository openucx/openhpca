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
	FileName       = "openhpca_results.txt"
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

/*
func String(dataDir string) (string, error) {
	metrics, err := ComputeScore(dataDir)
	if err != nil {
		return 0, unit, err
	}
	lines := strings.Split(string(content), "\n")
	for idx, line := range lines {
		if idx == 0 {
			// Skip the first line
			continue
		}

		if line == "" || strings.HasPrefix(line, "#") {
			if strings.Contains(line, "Latency (") {
				tokens := strings.Split(line, "Latency (")
				idx := strings.Index(tokens[1], ")")
				unit = tokens[1][:idx]
			}
			continue
		}

		// Parse a real data line
		words := strings.Split(line, " ")
		for _, w := range words {
			if w == "" || w == " " {
				continue
			}
			if size == -1.0 {
				size, err = strconv.ParseFloat(w, 32)
				if err != nil {
					return 0, unit, fmt.Errorf("unable to get actual latency data from %s: %w", w, err)
				}
			} else {
				lat, err = strconv.ParseFloat(w, 32)
				if err != nil {
					return 0, unit, fmt.Errorf("unable to get actual latency data from %s: %w", w, err)
				}
				return float32(lat), unit, nil
			}
		}
	}

	return 0, unit, fmt.Errorf("unable to find result file for latency")
}
*/

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

func (s *Metrics)Save(path string) error {
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