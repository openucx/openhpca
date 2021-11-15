//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package util

import (
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
