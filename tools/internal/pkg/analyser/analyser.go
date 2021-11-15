//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package analyser

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"

	osuresults "github.com/gvallee/go_osu/pkg/results"
	"github.com/gvallee/go_util/pkg/util"
	"github.com/gvallee/openhpca/tools/internal/pkg/score"
	"github.com/gvallee/validation_tool/pkg/label"
)

type Data struct {
	Text []string
	Unit string
}

type Results struct {
	resultsDir         string
	osuResultFiles     map[string]string
	smbResultFiles     map[string]string
	overlapResultFiles map[string]string
	osuFilesMap        map[string]string
	osuData            map[string]*Data
	overlapFilesMap    map[string]string
	overlapData        map[string]*Data
}

func (r *Results) GetOverlapData() map[string][]string {
	res := make(map[string][]string)
	for benchName, f := range r.overlapResultFiles {
		content, err := ioutil.ReadFile(filepath.Join(r.resultsDir, f))
		if err != nil {
			return nil
		}
		lines := strings.Split(string(content), "\n")
		if len(lines) < 2 {
			log.Printf("%s has an invalid content", f)
			return nil
		}
		res[benchName] = lines

		// We also store the results in the result object
		if r.overlapData == nil {
			r.overlapData = make(map[string]*Data)
		}
		d := new(Data)
		d.Text = lines
		r.overlapData[benchName] = d
		if r.overlapFilesMap == nil {
			r.overlapFilesMap = make(map[string]string)
		}
		r.overlapFilesMap[benchName] = f
	}
	return res
}

// LoadResultsWithPrefix enables loading OSU-type results from a file.
// This is meant to be used to load data from results files of different variants/versions of the OSU benchmark
func (r *Results) LoadResultsWithPrefix(testPrefix string) map[string]*Data {
	res := make(map[string]*Data)
	for benchName, f := range r.osuResultFiles {
		if strings.HasPrefix(benchName, testPrefix) {
			if testPrefix == "osu" && strings.HasPrefix(benchName, "osu_noncontig_mem") {
				continue
			}

			subBenchmark := strings.TrimPrefix(benchName, testPrefix+"_")
			f = filepath.Join(r.resultsDir, f)
			content, err := ioutil.ReadFile(f)
			if err != nil {
				return nil
			}
			text := string(content)
			lines := strings.Split(text, "\n")
			d := new(Data)
			d.Text = lines
			benchKey := testPrefix + "_" + subBenchmark
			res[benchKey] = d

			// We also store the results in the result object
			if r.osuData == nil {
				r.osuData = make(map[string]*Data)
			}
			if r.osuData[benchKey] == nil {
				d := new(Data)
				d.Text = lines
				if subBenchmark == "latency" {
					for _, line := range lines {
						if strings.HasPrefix(line, "# Size Latency (") {
							unit := strings.TrimPrefix(line, "# Size Latency (")
							unit = strings.TrimLeft(unit, "\n")
							unit = strings.TrimSuffix(unit, ")")
							d.Unit = unit
						}
					}
				}
				if subBenchmark == "bw" {
					for _, line := range lines {
						if strings.HasPrefix(line, "# Size Bandwidth (") {
							unit := strings.TrimPrefix(line, "# Size Bandwidth (")
							unit = strings.TrimLeft(unit, "\n")
							unit = strings.TrimSuffix(unit, ")")
							d.Unit = unit
						}
					}
				}
				r.osuData[benchKey] = d
			}
			if r.osuFilesMap == nil {
				r.osuFilesMap = make(map[string]string)
			}

			r.osuFilesMap[benchKey] = f
		}
	}

	return res
}

func getOutputFiles(dir string) (map[string]string, error) {
	outputFiles := make(map[string]string)
	allFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, file := range allFiles {
		filename := file.Name()
		if strings.HasSuffix(filename, ".out") {
			tokens := strings.Split(filename, "-")
			if len(tokens) == 3 {
				outputFiles[tokens[0]] = filename
			}
		}
	}
	return outputFiles, nil
}

func GetResults(dir string) (*Results, error) {
	r := new(Results)
	r.osuResultFiles = make(map[string]string)
	r.smbResultFiles = make(map[string]string)
	r.overlapResultFiles = make(map[string]string)
	r.resultsDir = dir

	if r.resultsDir == "" {
		return nil, fmt.Errorf("undefined result directory")
	}

	labelFile := label.GetFilePath(r.resultsDir)
	labels := make(map[string]string)
	err := label.FromFile(labelFile, labels)
	if err != nil {
		return nil, fmt.Errorf("label.FromFile() failed: %w", err)
	}

	outputFiles, err := getOutputFiles(dir)
	if err != nil {
		return nil, err
	}

	for hash, expLabel := range labels {
		filePath := outputFiles[hash]
		if filePath == "" {
			return nil, fmt.Errorf("unable to find output file for %s", expLabel)
		}

		if strings.HasPrefix(expLabel, "osu") {
			// OSU-type output file
			r.osuResultFiles[expLabel] = filePath
		}

		if strings.HasPrefix(expLabel, "smb") {
			// SMB-type output file
			r.smbResultFiles[expLabel] = filePath
		}

		if strings.HasPrefix(expLabel, "overlap") {
			// overlap-type output file
			expLabel = strings.ReplaceAll(expLabel, "overlap_overlap_", "overlap_")
			r.overlapResultFiles[expLabel] = filePath
		}
	}

	return r, nil
}

func (r *Results) GetLatency() (float32, string, error) {
	unit := "Unknown"
	size := -1.0
	lat := -1.0
	f := r.osuResultFiles["osu_latency"]
	if f == "" {
		return 0, "", fmt.Errorf("unable to get output file for OSU latency")
	}

	f = filepath.Join(r.resultsDir, f)
	content, err := ioutil.ReadFile(f)
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

func (r *Results) GetSMBOverlap() (float32, error) {
	f := r.smbResultFiles["smb_mpi_overhead"]
	if f == "" {
		return 0, fmt.Errorf("unable to get output file for SMB MPI overhead")
	}

	f = filepath.Join(r.resultsDir, f)
	content, err := ioutil.ReadFile(f)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) != 4 {
		return 0, fmt.Errorf("content of %s is not compliant with SMB mpi_overhead format (%d lines instead of 4)", f, len(lines))
	}
	targetLine := lines[2]
	for {
		// Cleanup the line to make parsing more reliable
		if !strings.Contains(targetLine, "  ") {
			break
		}
		targetLine = strings.ReplaceAll(targetLine, "  ", " ")
	}
	words := strings.Split(targetLine, " ")
	if len(words) != 8 {
		return 0, fmt.Errorf("invalid format: %s, %d elements instead of 8: %s", lines[2], len(words), words)
	}
	value, err := strconv.ParseFloat(words[6], 32)
	if err != nil {
		return 0, fmt.Errorf("unable to get actual data from %s: %w", words[6], err)
	}
	return float32(value), nil
}

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

func (r *Results) Plot(outputDir string) error {
	for benchmarkName, benchmarkData := range r.osuData {
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
			return fmt.Errorf("inconsistent data for %s: x axis has %d points while y axis has %d points (file: %s)", benchmarkName, len(x), len(y), r.osuFilesMap[benchmarkName])
		}

		err = PlotBenchmarkGraph(outputDir, benchmarkName, x, y)
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanOSUline(line string) string {
	line = strings.ReplaceAll(line, "\t", " ")
	for {
		if !strings.Contains(line, "  ") {
			break
		}
		line = strings.ReplaceAll(line, "  ", " ")
	}
	return line
}

func GetBandwidth(d *Data) (float64, string, error) {
	if d == nil {
		return 0, "", fmt.Errorf("undefined data")
	}

	unit := "Unknown"
	header := d.Text[2]
	header = cleanOSUline(header)
	unit = strings.TrimPrefix(header, "# Size Bandwidth (")
	unit = strings.TrimSuffix(unit, ")")

	idx := len(d.Text) - 1
	for {
		if d.Text[idx] == "" {
			idx--
		} else {
			break
		}
	}
	if !strings.HasPrefix(d.Text[idx], "4194304") {
		return -1, unit, fmt.Errorf("unexpected data, unable to find result for 4M messages: %s (idx: %d)", d.Text[idx], idx)
	}
	bw := 0.0
	for i := 0; i <= 5; i++ {
		line := d.Text[idx-i]
		line = cleanOSUline(line)
		tokens := strings.Split(line, " ")
		if len(tokens) != 2 {
			return -1, unit, fmt.Errorf("unexpected data, unable to find size and data: %s (idx: %d, len: %d)", line, idx, len(tokens))
		}
		tokens[1] = strings.TrimLeft(tokens[1], " ")
		tokens[1] = strings.TrimLeft(tokens[1], "\t")
		val, err := strconv.ParseFloat(tokens[1], 32)
		if err != nil {
			return -1, unit, fmt.Errorf("unexpected data, unable to parse data: %s (idx: %d)", d.Text[idx], idx)
		}
		if bw < val {
			bw = val
		}
	}

	if unit == "MB/s" {
		bw = bw / 125
		unit = "Gb/s"
	}

	return bw, unit, nil
}

func ComputeScore(dataDir string) (*score.Metrics, error) {
	data, err := GetResults(dataDir)
	if err != nil {
		return nil, err
	}

	osuData := data.LoadResultsWithPrefix("osu")
	mpiOverhead, err := data.GetSMBOverlap()
	if err != nil {
		return nil, err
	}
	bwData := osuData["bw"]
	bandwidth, bandwidthUnit, err := GetBandwidth(bwData)
	if err != nil {
		return nil, err
	}
	latency, latencyUnit, err := data.GetLatency()
	if err != nil {
		return nil, err
	}

	metrics := new(score.Metrics)
	metrics.Bandwidth = bandwidth
	metrics.BandwidthUnit = bandwidthUnit
	metrics.Latency = float64(latency)
	metrics.LatencyUnit = latencyUnit
	metrics.Overlap = float64(mpiOverhead)

	if bandwidthUnit != "Gb/s" {
		return nil, fmt.Errorf("unsupported unit for bandwidth (%s)", bandwidthUnit)
	}
	if latencyUnit != "us" {
		return nil, fmt.Errorf("unsupported unit for latency (%s)", latencyUnit)
	}

	metrics.Score = metrics.Compute()
	return metrics, nil
}
