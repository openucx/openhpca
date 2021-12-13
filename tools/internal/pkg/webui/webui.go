//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package webui

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"text/template"

	"github.com/gvallee/go_util/pkg/util"
	"github.com/openucx/openhpca/tools/internal/pkg/analyser"
	"github.com/openucx/openhpca/tools/internal/pkg/config"
	"github.com/openucx/openhpca/tools/internal/pkg/result"
)

const (
	DefaultPort     = 8080
	bwMetricID      = "osu_bw"
	latencyMetricID = "osu_latency"
)

type Config struct {
	Port              int
	indexTemplatePath string
	sourceCodeDir     string
	openhpcaCfg       *config.Data
}

type indexPageData struct {
	OSUData        map[string][]string
	Bandwidth      string
	BandwidthUnit  string
	Overlap        string
	OverlapUnit    string
	Latency        float32
	LatencyUnit    string
	OverlapData    map[string][]string
	OverlapDetails map[string]float32
	ScratchPath    string
	Score          int
}

type Server struct {
	mux                 *http.ServeMux
	indexTemplate       *template.Template
	cfg                 *Config
	httpServer          *http.Server
	wg                  *sync.WaitGroup
	data                *analyser.Results
	mpiOverhead         float32
	latency             float32
	latencyUnit         string
	bandwidth           float64
	bandwidthUnit       string
	osuData             map[string]*analyser.Data
	osuNonContigMemData map[string]*analyser.Data
	smbData             map[string][]string
	overlapData         map[string][]string
	overlapDetails      map[string]float32
	ipd                 indexPageData
}

func (c *Config) getTemplateFilePath(name string) string {
	return filepath.Join(c.sourceCodeDir, "tools", "internal", "pkg", "webui", "templates", name+".html")
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method, r.URL.String())
	s.mux.ServeHTTP(w, r)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	s.indexTemplate.Execute(w, s.ipd)
}

func (c *Config) newServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
		cfg: c,
	}

	s.mux.HandleFunc("/", s.index)
	s.mux.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir(s.cfg.openhpcaCfg.WP.ScratchDir))))

	s.indexTemplate = template.Must(template.New("index.html").Funcs(template.FuncMap{
		"getListResults": func(osuData map[string][]string, overlapData map[string][]string) string {
			content := ""
			for subbenchmark := range osuData {
				content += "\t\t\t\t<input type=\"radio\" name=\"operation\" class=\"tablinks\" value=\"" + subbenchmark + "\" onclick=\"openTab(event, '" + subbenchmark + "')\"/>\n"
				content += "\t\t\t\t<label for=\"" + subbenchmark + "\">" + subbenchmark + "</label><br/>\n"
			}
			for subbenchmark := range overlapData {
				content += "\t\t\t\t<input type=\"radio\" name=\"operation\" class=\"tablinks\" value=\"" + subbenchmark + "\" onclick=\"openTab(event, '" + subbenchmark + "')\"/>\n"
				content += "\t\t\t\t<label for=\"" + subbenchmark + "\">" + subbenchmark + "</label><br/>\n"
			}
			return content
		},
		"getResultDetails": func(osuData map[string][]string, overlapData map[string][]string, scratchPath string) string {
			content := ""
			for subbenchmark, results := range osuData {
				content += "<div id=\"" + subbenchmark + "\" class=\"tabcontent\">"
				for _, line := range results {
					content += line + "<br/>\n"
				}
				plotPath := filepath.Join(scratchPath, subbenchmark+".png")
				if util.FileExists(plotPath) {
					content += "<img src=\"images/" + subbenchmark + ".png\" />\n"
				}
				content += "</div>\n"
			}
			for subbenchmark, results := range overlapData {
				content += "<div id=\"" + subbenchmark + "\" class=\"tabcontent\">"
				for _, line := range results {
					content += line + "<br/>\n"
				}
				content += "</div>\n"
			}
			return content
		},
		"getListMainResults": func(osuData map[string][]string, overlapData map[string][]string) string {
			content := ""
			for subbenchmark := range osuData {
				if subbenchmark == bwMetricID || subbenchmark == latencyMetricID {
					label := ""
					if subbenchmark == bwMetricID {
						label = "Bandwidth"
					}
					if subbenchmark == latencyMetricID {
						label = "Latency"
					}
					content += "\t\t\t\t<input type=\"radio\" name=\"operation\" class=\"tablinks\" value=\"" + subbenchmark + "\" id=\"" + subbenchmark + "_button\" onclick=\"openTab(event, '" + subbenchmark + "')\"/>\n"
					content += "\t\t\t\t<label for=\"" + subbenchmark + "\">" + label + "</label><br/>\n"
				}
			}
			content += "\t\t\t\t<input type=\"radio\" name=\"operation\" class=\"tablinks\" value=\"overlap\" onclick=\"openTab(event, 'overlap')\"/>\n"
			content += "\t\t\t\t<label for=\"overlap\">Overlap</label><br/>\n"

			return content
		},
		"getResultMainDetails": func(osuData map[string][]string, overlapDetails map[string]float32, overlapScore string, scratchPath string) string {
			content := ""
			for subbenchmark, results := range osuData {
				if subbenchmark == bwMetricID || subbenchmark == latencyMetricID {
					content += "<div id=\"" + subbenchmark + "\" class=\"tabcontent\">"
					for _, line := range results {
						content += line + "<br/>\n"
					}
					plotPath := filepath.Join(scratchPath, subbenchmark+".png")
					if util.FileExists(plotPath) {
						content += "<img src=\"images/" + subbenchmark + ".png\" />\n"
					}
					content += "</div>\n"
				}
			}
			content += "<div id =\"overlap\" class=\"tabcontent\">"
			content += fmt.Sprintf("Overlap score: %s %% <br/><br/>Details:<br/>", overlapScore)
			for name, score := range overlapDetails {
				content += fmt.Sprintf("%s score: %.1f</br>\n", name, score)
			}
			content += "</div>"
			return content
		},
		"displaySelection": func(osuData map[string][]string, overlapData map[string][]string, zone string) string {
			content := ""
			if zone == "left" {
				content += "<select id=\"select_for_comp_left\" onChange=\"updateLeftSelectionForComp()\">"
			} else {
				content += "<select id=\"select_for_comp_right\" onChange=\"updateRightSelectionForComp()\">"
			}
			for subbenchmark := range osuData {
				content += "\t\t\t\t<option value=\"" + subbenchmark + "\">" + subbenchmark + "</option>\n"
			}
			for subbenchmark := range overlapData {
				content += "\t\t\t\t<option value=\"" + subbenchmark + "\">" + subbenchmark + "</option>\n"
			}
			content += "</select>\n"
			return content
		},
		"getCompBenchmarkDetailsLeft": func(osuData map[string][]string, overlapData map[string][]string, scratchPath string) string {
			content := ""
			for subbenchmark, results := range osuData {
				content += "<div id=\"" + subbenchmark + "_comp_data_left\" class=\"compdataleft\">"
				for _, line := range results {
					content += line + "<br/>\n"
				}
				plotPath := filepath.Join(scratchPath, subbenchmark+".png")
				if util.FileExists(plotPath) {
					content += "<img src=\"images/" + subbenchmark + ".png\" />\n"
				}
				content += "</div>\n"
			}
			for subbenchmark, results := range overlapData {
				content += "<div id=\"" + subbenchmark + "_comp_data_left\" class=\"compdataleft\">"
				for _, line := range results {
					content += line + "<br/>\n"
				}
				content += "</div>\n"
			}
			return content
		},
		"getCompBenchmarkDetailsRight": func(osuData map[string][]string, overlapData map[string][]string, scratchPath string) string {
			content := ""
			for subbenchmark, results := range osuData {
				content += "<div id=\"" + subbenchmark + "_comp_data_right\" class=\"compdataright\">"
				for _, line := range results {
					content += line + "<br/>\n"
				}
				plotPath := filepath.Join(scratchPath, subbenchmark+".png")
				if util.FileExists(plotPath) {
					content += "<img src=\"images/" + subbenchmark + ".png\" />\n"
				}
				content += "</div>\n"
			}
			for subbenchmark, results := range overlapData {
				content += "<div id=\"" + subbenchmark + "_comp_data_right\" class=\"compdataright\">"
				for _, line := range results {
					content += line + "<br/>\n"
				}
				content += "</div>\n"
			}
			return content
		},
	}).ParseFiles(s.cfg.indexTemplatePath))

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: s,
	}

	s.wg = new(sync.WaitGroup)
	s.wg.Add(1)

	return s
}

func Init(verbose bool) (*Config, error) {
	cfg := new(Config)
	_, filename, _, _ := runtime.Caller(0)
	cfg.sourceCodeDir = filepath.Join(filepath.Dir(filename), "..", "..", "..", "..")
	cfg.indexTemplatePath = cfg.getTemplateFilePath("index")
	cfg.Port = DefaultPort
	cfg.openhpcaCfg = new(config.Data)
	cfg.openhpcaCfg.Basedir = cfg.sourceCodeDir
	cfg.openhpcaCfg.BinName = filename

	// Load the OpenHPCA configuration
	err := cfg.openhpcaCfg.Load()
	if err != nil {
		fmt.Printf("Unable to load OpenHPCA configuration: %s\n", err)
		os.Exit(1)
	}

	return cfg, nil
}

// Start instantiates a HTTP server and start the webUI. This is a non-blocking function,
// meaning the function returns after successfully initiating the WebUI. To wait for the
// termination of the webUI, please use Wait()
func (c *Config) Start() (*Server, error) {
	var err error
	s := c.newServer()
	s.data, err = analyser.GetResults(c.openhpcaCfg.GetRunDir())
	if err != nil {
		return nil, err
	}

	s.osuData = s.data.LoadResultsWithPrefix("osu")
	if len(s.osuData) == 0 {
		return nil, fmt.Errorf("no OSU data")
	}
	s.osuNonContigMemData = s.data.LoadResultsWithPrefix("osu_noncontig_mem")
	if len(s.osuNonContigMemData) == 0 {
		return nil, fmt.Errorf("no OSU data for non-contiguous memory")
	}

	s.mpiOverhead, err = s.data.GetSMBOverlap()
	if err != nil {
		return nil, err
	}
	s.latency, s.latencyUnit, err = s.data.GetLatency()
	if err != nil {
		return nil, err
	}

	bwData := s.osuData[bwMetricID]
	if bwData == nil {
		return nil, fmt.Errorf("undefined bandwidth data")
	}
	s.bandwidth, s.bandwidthUnit, err = analyser.GetBandwidth(bwData)
	if err != nil {
		return nil, err
	}

	s.ipd.OSUData = make(map[string][]string)
	for key, val := range s.osuData {
		s.ipd.OSUData[key] = val.Text
	}
	for key, val := range s.osuNonContigMemData {
		s.ipd.OSUData[key] = val.Text
	}

	s.overlapData = s.data.GetOverlapData()
	var overlapScore float32
	overlapScore, s.ipd.OverlapDetails, err = result.ComputeOverlap(s.mpiOverhead, s.overlapData)
	if err != nil {
		return nil, err
	}
	s.ipd.Overlap = fmt.Sprintf("%.0f", overlapScore)
	s.ipd.Bandwidth = fmt.Sprintf("%.2f", s.bandwidth)
	s.ipd.BandwidthUnit = s.bandwidthUnit
	s.ipd.Latency = s.latency
	s.ipd.LatencyUnit = s.latencyUnit
	s.ipd.ScratchPath = s.cfg.openhpcaCfg.WP.ScratchDir

	/*
		metrics := new(score.Metrics)
		metrics.Bandwidth = s.ipd.Bandwidth
		metrics.Latency = float64(s.ipd.Latency)
		metrics.Overlap = float64(s.ipd.Overlap)
		s.ipd.Score = metrics.Compute()
	*/

	s.ipd.OverlapData = s.overlapData
	err = s.data.Plot(s.cfg.openhpcaCfg.WP.ScratchDir)
	if err != nil {
		return nil, err
	}

	go func(s *Server) {
		defer s.wg.Done()
		err := s.httpServer.ListenAndServe()
		if err != nil {
			fmt.Printf("WebServer internal error: %s\n", err)
		}
		fmt.Println("HTTP server is now terminated")
	}(s)

	return s, nil
}

// Wait makes the current process wait for the termination of the webUI
func (s *Server) Wait() {
	s.wg.Wait()
}
