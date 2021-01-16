package chingwei

import (
	docker "github.com/fsouza/go-dockerclient"
)

func RunMysql(version string) error {
	return nil
}
func PrepareMySQL(version string) (DBConnInfo, error) {
	// fetch mysql container
	client, newClientErr := docker.NewClientFromEnv()
	if newClientErr != nil {
		panic(newClientErr)
	}

	filter := make(map[string]string)
	filter["ancestor"] = "mysql-server"

	// should only yield one container
	containers, listErr := client.ListContainers(docker.ListContainersOptions{All: false})
	if listErr != nil {
		panic(listErr)
	}
	if len(containers) != 1 {

	}

	// connect to mysql and run

	return DBConnInfo{Host: "", Port: "3306", User: "root", Database: "test"}, nil
}
