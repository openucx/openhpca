//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package download

import "fmt"

type DownloaderCfg struct {
	// Type is the identifier of the downloader to use on the system
	Type int

	// Bin is the path to the downloader binary
	Bin string

	// Args is the list of arguments to use with the binary to download software
	Args []string
}

type URLFn func(*DownloaderCfg, string, string) error

type Downloader struct {
	cfg           DownloaderCfg
	internalFnURL URLFn
}

func (d *Downloader) Load() error {
	// First try to load the wget downloader
	err := d.WgetLoad()
	if err == nil {
		// wget can be used, all done
		return nil
	}

	return fmt.Errorf("unable to detect any software to download packages")
}

func (d *Downloader) Init() error {
	err := d.Load()
	if err != nil {
		return err
	}
	return nil
}

// URL downloads a give URL into a target directory
func (d *Downloader) URL(url string, destDir string) error {
	return d.internalFnURL(&d.cfg, url, destDir)
}
