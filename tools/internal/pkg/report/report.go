//
// Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package report

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/openucx/openhpca/tools/internal/pkg/config"
	"github.com/openucx/openhpca/tools/internal/pkg/runErrors"
)

const (
	errorFileSuffix = ".err"
)

func addConfigFileContent(cfg *config.Data, reportFile *os.File) error {
	_, err := reportFile.Write([]byte("# Configuration\n\n"))
	if err != nil {
		return fmt.Errorf("unable to write header for the content of the configuration file: %w", err)
	}

	// Copy the content of the configuration file
	configData, err := os.ReadFile(cfg.WP.ConfigFile)
	if err != nil {
		return fmt.Errorf("unable to read the content of the configuration file %s: %w", cfg.ConfigFile, err)
	}
	_, err = reportFile.Write(configData)
	if err != nil {
		return fmt.Errorf("unable to copy the content of the configuration file into the report file: %w", err)
	}

	_, err = reportFile.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("unable to write footer for the content of the configuration file: %w", err)
	}

	return nil
}

func saveUnknownErrors(cfg *config.Data, unknownErrors []string) error {
	if len(unknownErrors) == 0 {
		return nil
	}
	unknownErrorLogFilePath := filepath.Join(cfg.WP.Basedir, "unknown_errors.md")
	unknownErrorLogFile, err := os.Create(unknownErrorLogFilePath)
	if err != nil {
		return fmt.Errorf("unable to create unknown error log file %s: %w", unknownErrorLogFilePath, err)
	}
	defer unknownErrorLogFile.Close()

	for idx, errMsg := range unknownErrors {
		msg := string(errMsg)
		_, err = unknownErrorLogFile.Write([]byte(fmt.Sprintf("# Error %d:\n%s\n\n", idx, msg)))
		if err != nil {
			return fmt.Errorf("unable to add error message to unknown error log: %w, %s", err, errMsg)
		}
	}

	fmt.Printf("Successfully create %s\n", unknownErrorLogFilePath)

	return nil
}

func saveKnownErrors(cfg *config.Data, reportFile *os.File, numErrors map[int]int, errorMsgs map[int][]string) error {
	if len(numErrors) == 0 {
		return nil
	}

	_, err := reportFile.Write([]byte("# Catalogued errors\n\n"))
	if err != nil {
		return fmt.Errorf("unable to write the catalogued errors header: %w", err)
	}

	for errIdx, count := range numErrors {
		reportFile.Write([]byte(fmt.Sprintf("%d error of type:\n%s\n\n", count, runErrors.KnownErrors[errIdx])))
		for _, msg := range errorMsgs[errIdx] {
			_, err := reportFile.Write([]byte(fmt.Sprintf("%s\n\n", msg)))
			if err != nil {
				return fmt.Errorf("unable to add error message %s to report: %w", msg, err)
			}
		}
	}
	return nil
}

func analyzeRunErrors(cfg *config.Data, reportFile *os.File) error {
	var errorFiles []string
	runDir := cfg.GetRunDir()

	d, err := ioutil.ReadDir(runDir)
	if err != nil {
		return fmt.Errorf("unable to get list of files in %s: %w", runDir, err)
	}

	for _, entry := range d {
		if strings.HasSuffix(entry.Name(), errorFileSuffix) {
			errorFiles = append(errorFiles, filepath.Join(runDir, entry.Name()))
		}
	}

	cataloguedErrors := make(map[int][]string)
	numKnownErrors := make(map[int]int)
	var unknownErrors []string
	successfulRuns := 0
	failedRuns := 0
	for _, errorFilePath := range errorFiles {
		// Is the file empty?
		s, err := os.Stat(errorFilePath)
		if err != nil {
			return fmt.Errorf("unable to get statistics about %s: %w", errorFilePath, err)
		}
		if s.Size() != 0 {
			failedRuns++
			// Get the error message from error log
			errorMsgData, err := os.ReadFile(errorFilePath)
			if err != nil {
				return fmt.Errorf("unable to read file %s: %w", errorFilePath, err)
			}
			errIdx := runErrors.IsKnownError(string(errorMsgData))
			if errIdx >= 0 {
				cataloguedErrors[errIdx] = append(cataloguedErrors[errIdx], string(errorMsgData))
				if _, ok := numKnownErrors[errIdx]; !ok {
					numKnownErrors[errIdx] = 1
				} else {
					numKnownErrors[errIdx]++
				}
			} else {
				unknownErrors = append(unknownErrors, string(errorMsgData))
			}
		} else {
			successfulRuns++
		}
	}

	// Save overall count of successful and failed runs
	_, err = reportFile.Write([]byte(fmt.Sprintf("# Results overview\n\nNumber of successful runs: %d\nNumber of failed runs: %d\n\n", successfulRuns, failedRuns)))
	if err != nil {
		return fmt.Errorf("unable to save the results overview into the report")
	}

	// Save all the error message data in a separate file
	err = saveUnknownErrors(cfg, unknownErrors)
	if err != nil {
		return fmt.Errorf("saveUnknownErrors() failed; unable to save the detected unknown errors: %w", err)
	}

	// Save the known issues
	err = saveKnownErrors(cfg, reportFile, numKnownErrors, cataloguedErrors)
	if err != nil {
		return fmt.Errorf("saveKnownErrors() failed; unalble to save the detected known error: %w", err)
	}

	return nil
}

func Generate(cfg *config.Data) error {
	reportFilePath := filepath.Join(cfg.WP.Basedir, "report.md")
	reportFile, err := os.Create(reportFilePath)
	if err != nil {
		return fmt.Errorf("unable to create report file %s: %w", reportFilePath, err)
	}
	defer reportFile.Close()

	err = addConfigFileContent(cfg, reportFile)
	if err != nil {
		return fmt.Errorf("unable to add the content of the configuration file: %w", err)
	}

	err = analyzeRunErrors(cfg, reportFile)
	if err != nil {
		return fmt.Errorf("unable to analyze the run errors: %w", err)
	}

	fmt.Printf("Successfully create %s\n", reportFilePath)

	return nil
}
