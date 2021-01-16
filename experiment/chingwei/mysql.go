package chingwei

func RunMysql(version string) error {
	return nil
}
func PrepareMySQL(version string) (DBConnInfo, error) {
	// fetch mysql container
	client, err := docker.NewClientFromEnv()
	if err != nil {
		panic(err)
	}

	// connect to mysql and run

	return DBConnInfo{Host: "", Port: "3306", User: "root", Database: "test"}, nil
}
