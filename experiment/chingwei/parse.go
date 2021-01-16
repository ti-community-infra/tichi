package chingwei

import (
	"regexp"
	"strings"

	"k8s.io/test-infra/prow/github"
)

func parseIssue(issue github.Issue) (query string, mysqlVersion string, tidbVersion string, expected string, actual string) {
	re := regexp.MustCompile(`### Steps to reproduce
(?P<query>(.+\n)+)
### What is expected\?
(?P<expected>(.+\n)+)
### What is actually happening\?
(?P<actual>(.+\n)+)
\| Environment \| Info \|
\|---\|---\|
\| TiDB Version \| (?P<tidbVersion>(v\d|.+)) \|
\| MySQL Version \| (?P<mysqlVersion>(\d|.+)) \|`)

	body := issue.Body
	matches := re.FindStringSubmatch(body)

	query = matches[re.SubexpIndex("query")]
	expected = matches[re.SubexpIndex("expected")]
	actual = matches[re.SubexpIndex("actual")]
	tidbVersion = matches[re.SubexpIndex("tidbVersion")]
	mysqlVersion = matches[re.SubexpIndex("mysqlVersion")]

	query = strings.TrimSpace(query)
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	tidbVersion = strings.TrimSpace(tidbVersion)
	mysqlVersion = strings.TrimSpace(mysqlVersion)

	return
}
