//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package score

import "fmt"

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

func (s *Metrics) Compute() int {
	return -1
	//return int(100 - 1000/s.Bandwidth + 100 - 100 * float64(s.Latency) + float64(s.Overlap))
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
