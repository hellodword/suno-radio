package common

import "os/exec"

func CheckFfmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}
