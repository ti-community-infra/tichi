package chingwei

import (
	"regexp"
	"strings"

	"k8s.io/test-infra/prow/github"
)

func parseIssue(issue github.Issue) *IssueBasicInfo {
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
	query := strings.TrimSpace(matches[re.SubexpIndex("query")])
	expected := strings.TrimSpace(matches[re.SubexpIndex("expected")])
	actual := strings.TrimSpace(matches[re.SubexpIndex("actual")])
	tidbVersion := strings.TrimSpace(matches[re.SubexpIndex("tidbVersion")])
	mysqlVersion := strings.TrimSpace(matches[re.SubexpIndex("mysqlVersion")])

	return &IssueBasicInfo{
		query:        query,
		expected:     expected,
		actual:       actual,
		tidbVersion:  tidbVersion,
		mysqlVersion: mysqlVersion,
	}
}
