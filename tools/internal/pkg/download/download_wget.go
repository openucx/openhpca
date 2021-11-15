//
// Copyright (c) 2020-2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package download

import "os/exec"

func wgetURL(cfg *DownloaderCfg, url string, destDir string) error {
	args := append(cfg.Args, url)
	cmd := exec.Command(cfg.Bin, args...)
	cmd.Dir = destDir
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (d *Downloader) WgetLoad() error {
	binPath, err := exec.LookPath("wget")
	if err != nil {
		return err
	}
	d.cfg.Bin = binPath
	d.cfg.Args = nil
	d.internalFnURL = wgetURL
	return nil
}
