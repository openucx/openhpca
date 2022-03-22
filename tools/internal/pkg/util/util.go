//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package util

import (
	"io/ioutil"
	"strings"
)

const (
	// OpenHPCADirID is the string used in the configuration for implicit substitution
	// with the location of OpenHPCA's source code at execution time
	OpenHPCADirID = "OPENHPCA_DIR"
)

// UpdateOpenHPCADirValue replaces OPENHPCA_DIR by the actual value of where the OpenHPCA code sits
func UpdateOpenHPCADirValue(str string, basedir string) string {
	return strings.ReplaceAll(str, OpenHPCADirID, basedir)
}

func CleanOSUline(line string) string {
	line = strings.ReplaceAll(line, "\t", " ")
	for {
		if !strings.Contains(line, "  ") {
			break
		}
		line = strings.ReplaceAll(line, "  ", " ")
	}
	return line
}

func GetOutputFiles(dir string) (map[string]string, error) {
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
