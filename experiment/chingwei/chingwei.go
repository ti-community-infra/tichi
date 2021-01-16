package chingwei

import (
	"github.com/sirupsen/logrus"
	"io"
	"k8s.io/test-infra/prow/github"
	"log"
	"os"
	"os/exec"
	"strings"
)

type githubClient interface {
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
}

func Reproducing(log *logrus.Entry, ghc githubClient) error {
	log.Info("Staring search pull request.")
	//filter := `repo:"tidb" label:"status/needs-reproduction"`
	//
	//issues, err := ghc.FindIssues(filter, "", false)
	//if err != nil {
	//	return err
	//}

	// only reproduce first issue
	//issue := issues[0]
	// parse minimal reproduce step and version from issue
	//query, mysqlVersion, tidbVersion, expected, actual := parseIssue(issue)

	// try send a SQL query to tidb with a specific version
	//tidbInfo, err := PrepareTiDB(tidbVersion)
	mysqlInfo, err := PrepareMySQL("8.0.21")
	if err != nil {
		return err
	}

	// reproduce by connecting to tidb and mysql
	//tidbOutput, tidbErr := Reproduce(tidbInfo, query)
	mysqlOutput, mysqlErr := Reproduce(*mysqlInfo, "show status;")

	//if tidbErr != nil {
	//	panic(tidbErr)
	//} else

	if mysqlErr != nil {
		panic(mysqlErr)
	}

	println(mysqlOutput)
	// Feedback to issue.
	// diff expected v.s. mysqlOutput
	// diff actual v.s. tidbOutput into folded section
	//_ = DiffSubmittedAndExecuted(expected, mysqlOutput)
	//_ = DiffSubmittedAndExecuted(actual, tidbOutput)

	// compare result
	//_ = CompareResult(mysqlOutput, tidbOutput)

	return nil
}

type DBConnInfo struct {
	Host     string
	Port     string
	User     string
	Database string
	Password string
}

func Reproduce(dbconninfo DBConnInfo, query string) (string, error) {
	// create the query file
	f, ferr := os.Create("query.sql")
	if ferr != nil {
		log.Println("Cannot create query file for mysql.")
		panic(ferr)
	}
	defer f.Close()
	_, writeErr := f.WriteString(query)
	if writeErr != nil {
		log.Println("Cannot write query file for mysql.")
	}
	// execute the queries
	cmd := exec.Command("mysql",
		"-u", dbconninfo.User,
		"-p",
		"-D", dbconninfo.Database,
		"<", "query.sql")
	subStdin, err := cmd.StdinPipe()
	defer subStdin.Close()
	// input the password
	_, consoleError := io.WriteString(subStdin, dbconninfo.Password)
	if consoleError != nil {
		log.Fatalln()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Failed to run query")
		panic(err)
	}
	return string(out), nil
}

// mock diff
func DiffSubmittedAndExecuted(submitted string, executed string) string {
	return "no differences"
}

// mock simple comparison
func CompareResult(expected string, actual string) bool {
	return strings.Compare(expected, actual) == 0
}
