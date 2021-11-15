//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package score

import "fmt"

type Metrics struct {
	Bandwidth     float64
	BandwidthUnit string
	Latency       float64
	LatencyUnit   string
	Overlap       float64
	Score         int
}

func (s *Metrics) Compute() int {
	return -1
	//return int(100 - 1000/s.Bandwidth + 100 - 100 * float64(s.Latency) + float64(s.Overlap))
}

func (s *Metrics) ToString() string {
	content := fmt.Sprintf("Bandwidth: %f %s\n", s.Bandwidth, s.BandwidthUnit)
	content += fmt.Sprintf("Latency: %f %s\n", s.Latency, s.LatencyUnit)
	content += fmt.Sprintf("Overlap: %f %%\n", s.Overlap)
	content += "\n"
	//content += fmt.Sprintf("Score: %d\n", s.Score)
	return content
}
