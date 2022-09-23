//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package analyser

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"

	osuresults "github.com/gvallee/go_osu/pkg/results"
	"github.com/gvallee/go_util/pkg/util"
	"github.com/openucx/openhpca/tools/internal/pkg/result"
)


func preparePoints(n int, x []float64, y []float64) plotter.XYs {
	pts := make(plotter.XYs, n)
	for i := range pts {
		pts[i].X = x[i]
		pts[i].Y = y[i]
	}
	return pts
}

func PlotBenchmarkGraph(outputDir string, operationName string, x []float64, y []float64) error {
	filePath := filepath.Join(outputDir, operationName+".png")
	if !util.FileExists(filePath) {
		p := plot.New()

		p.Title.Text = operationName
		p.X.Label.Text = "Size"
		p.Y.Label.Text = "Avg Latency (us)"

		err := plotutil.AddLinePoints(p, operationName, preparePoints(len(x), x, y))
		if err != nil {
			return err
		}

		err = p.Save(4*vg.Inch, 4*vg.Inch, filePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func Plot(r *result.Data, outputDir string) error {
	for benchmarkName, benchmarkData := range r.OsuData {
		if strings.HasPrefix(benchmarkName, "i") || strings.Contains(benchmarkName, "_i") {
			log.Printf("skipping plotting of %s since it is a non-blocking operation", benchmarkName)
			continue
		}
		if strings.Contains(benchmarkName, "barrier") {
			continue
		}

		// extract the data from the benchmark output
		x, y, err := osuresults.ExtractDataFromOutput(benchmarkData.Text)
		if err != nil {
			return err
		}

		if len(x) == 0 {
			return fmt.Errorf("unable to extract data for %s", benchmarkName)
		}

		if len(x) != len(y) {
			return fmt.Errorf("inconsistent data for %s: x axis has %d points while y axis has %d points (file: %s)", benchmarkName, len(x), len(y), r.OsuFilesMap[benchmarkName])
		}

		err = PlotBenchmarkGraph(outputDir, benchmarkName, x, y)
		if err != nil {
			return err
		}
	}

	return nil
}



