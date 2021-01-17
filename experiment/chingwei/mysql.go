package chingwei

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func PrepareMySQL(version string) (*DBConnInfo, error) {
	startDocker := exec.Command("service", "docker", "start")
	err := startDocker.Run()
	if err != nil {
		return nil, err
	}

	// Waiting docker starting.
	time.Sleep(30 * time.Second)

	// Pull the image.
	containerName := "mysql/mysql-server:" + version
	password := "zhangyushao"
	cmd := exec.Command("docker", "run", "-d", "-e", "MYSQL_ROOT_PASSWORD="+password,
		"-e", "MYSQL_ROOT_HOST=%", "-p", "3306:3306", "--name", "mysqlcw",
		containerName, "--default-authentication-plugin=mysql_native_password")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Println("Failed to run MySQL container")
		return nil, err
	}

	host := "0.0.0.0"
	port := "3306"
	wait := exec.Command("/usr/local/bin/wait-for-it.sh", "-h", host, "-p", port, "-t", "0")
	err = wait.Run()
	if err != nil {
		return nil, fmt.Errorf("wait for mysql service failed: %w", err)
	}
	time.Sleep(30 * time.Second)
	log.Println("Started MySQL server.")

	// connect to mysql and run
	return &DBConnInfo{Host: "0.0.0.0",
		Port:     "3306",
		User:     "root",
		Database: "mysql",
		Password: password}, nil
}
