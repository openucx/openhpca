//
// Copyright (c) 2022, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package runErrors

import "strings"

const (
	pmixError = `--------------------------------------------------------------------------
	An ORTE daemon has unexpectedly failed after launch and before
	communicating back to mpirun. This could be caused by a number
	of factors, including an inability to create a connection back
	to mpirun due to a lack of common network interfaces and/or no
	route found between them. Please check network connectivity
	(including firewalls and network routing requirements).
	--------------------------------------------------------------------------`
	slurmTimeOut                     = "slurmstepd"
	overlapBenchElementIncreaseError = "Cannot further increase n_elts"
	overlapBenchCalibrationFailure   = "Calibration failed"
)

var KnownErrors []string = []string{pmixError, slurmTimeOut, overlapBenchElementIncreaseError, overlapBenchCalibrationFailure}

func IsKnownError(errorMsg string) int {
	for idx, e := range KnownErrors {
		if strings.Contains(errorMsg, e) {
			return idx
		}
	}
	return -1
}
