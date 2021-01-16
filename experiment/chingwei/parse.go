package chingwei

import "k8s.io/test-infra/prow/github"

func parseIssue(issue github.Issue) (query string, mysqlVersion string, tidbVersion string, expected string, actual string) {

	query = `
CREATE TABLE test (
	iD bigint(20) NOT NULL,
	INT_TEST int(11) DEFAULT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC;
INSERT INTO test VALUES (2, 10), (3, NULL);

SELECT DISTINCT count(*), id + int_test as res FROM test  GROUP BY res ORDER BY res;
`
	tidbVersion = "v4.0.0"
	return
}
