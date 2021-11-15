//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package score

import (
	"fmt"
	"testing"
)

func TestStringToFloatBase(t *testing.T) {
	tests := []struct {
		input          string
		base           float64
		expectedOutput float64
	}{
		{
			input:          "1",
			base:           10,
			expectedOutput: 1,
		},
		{
			input:          "1.1",
			base:           10,
			expectedOutput: 1.1,
		},
		{
			input:          "123.4",
			base:           10,
			expectedOutput: 123.4,
		},
		{
			input:          "1",
			base:           100,
			expectedOutput: 1,
		},
		{
			input:          "1.1",
			base:           100,
			expectedOutput: 1.01,
		},
		{
			input:          "123.45",
			base:           100,
			expectedOutput: 10203.0405,
		},
		{
			input:          "1000",
			base:           0.1,
			expectedOutput: 0.001,
		},
		{
			input:          "213",
			base:           0.1,
			expectedOutput: 3.12,
		},
		{
			input:          "213",
			base:           0.01,
			expectedOutput: 3.0102,
		},
		{
			input:          "20000",
			base:           0.1,
			expectedOutput: 0.0002,
		},
	}

	for _, tt := range tests {
		val, err := StringToFloatBase(tt.input, tt.base)
		if err != nil {
			t.Fatalf("StringToFloatBase() failed: %s", err)
		}
		valStr := fmt.Sprintf("%f", val)
		expectedStr := fmt.Sprintf("%f", tt.expectedOutput)
		if valStr != expectedStr {
			t.Fatalf("StringToFloatBase() returned %f instead of %f (input: %s, base: %f)", val, tt.expectedOutput, tt.input, tt.base)
		}
	}
}
