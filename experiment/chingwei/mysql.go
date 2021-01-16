package chingwei

import (
	//docker "github.com/fsouza/go-dockerclient"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func PrepareMySQL(version string) (*DBConnInfo, error) {
	//client, newClientErr := docker.NewClientFromEnv()
	//if newClientErr != nil {
	//	panic(newClientErr)
	//}

	// create mysql container
	//client.PullImage(docker.PullImageOptions{
	//	Repository: "mysql",
	//	Tag: version,
	//	Platform: version})
	//client.CreateContainer(docker.CreateContainerOptions{Name: "mysql"})

	//filter := make(map[string][]string)
	//filter["ancestor"] = []string{"mysql-server"}
	//
	//// should only yield one container
	//containers, listErr := client.ListContainers(docker.ListContainersOptions{All: false, Filters: filter})
	//if listErr != nil {
	//	panic(listErr)
	//}
	//if len(containers) != 1 {
	//	log := logrus.StandardLogger().WithField("component", "ching-wei")
	//	log.Warn("Detected more than one container is running")
	//}
	startDocker := exec.Command("service", "docker", "start")
	err := startDocker.Run()
	if err != nil {
		return nil, err
	}
	time.Sleep(30 * time.Second)
	containerName := "mysql/mysql-server:" + version
	password := "zhangyushao"
	cmd := exec.Command("docker", "run", "-d", "-e", "MYSQL_ROOT_PASSWORD="+password, "-e", "MYSQL_ROOT_HOST=%", "-p", "3306:3306", "--name", "mysqlcw", containerName, "--default-authentication-plugin=mysql_native_password")
	fmt.Println("docker command:", cmd)
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
