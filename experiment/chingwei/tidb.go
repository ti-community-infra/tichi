package chingwei

import (
	"fmt"
	"os"
	"os/exec"
)

// PrepareTiDB start a tidb server with specific version.
// It returns a function to destory and cleanup the related resources.
func PrepareTiDB(version string) (*DBConnInfo, error) {
	cmd := exec.Command("/root/.tiup/bin/tiup", "playground", version, "--tiflash", "0", "--monitor=false")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("tiup playground failed: %w", err)
	}
	host := "127.0.0.1"
	port := "4000"
	wait := exec.Command("/usr/local/bin/wait-for-it.sh", "-h", host, "-p", port, "-t", "0")
	err = wait.Run()
	if err != nil {
		return nil, fmt.Errorf("wait for tidb service failed: %w", err)
	}
	return &DBConnInfo{
		Host:     host,
		Port:     port,
		User:     "root",
		Database: "test",
	}, nil
}
