package chingwei

import (
	//docker "github.com/fsouza/go-dockerclient"
	"log"
	"os/exec"
	"strings"
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

	containerName := "mysql/mysql-server:" + version
	cmd := exec.Command("docker", "run", "-d", "--name", "mysqlcw", containerName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Failed to run MySQL container")
		return nil, err
	}

	// pattern match generated password
	output := string(out)
	outlines := strings.Split(output, "\n")
	password := ""
	found := false
	for _, line := range outlines {
		if strings.Contains(line, "GENERATED") {
			parts := strings.Split(line, ": ")
			password = strings.Replace(parts[1], " ", "", 10086)
			found = true
			break
		}
	}

	// incase the password is not generated
	if !found {
		log.Fatalln("Failed to capture initialize MySQL root password.", output)
	}

	// connect to mysql and run
	return &DBConnInfo{Host: "localhost",
		Port:     "3306",
		User:     "root",
		Database: "mysql",
		Password: password}, nil
}
