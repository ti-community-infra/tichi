package chingwei

type IssueBasicInfo struct {
	query        string
	expected     string
	actual       string
	tidbVersion  string
	mysqlVersion string
}

type DBConnInfo struct {
	Host     string
	Port     string
	User     string
	Database string
	Password string
}
