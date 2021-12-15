package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	issueNumberBlockRegexpTemplate  = "(?i)%s\\s*%s(?P<issue_number>[1-9]\\d*)"
	associatePrefixRegexp           = "(?P<associate_prefix>ref|close[sd]?|resolve[sd]?|fix(e[sd])?)"
	orgRegexp                       = "[a-zA-Z0-9][a-zA-Z0-9-]{0,38}"
	repoRegexp                      = "[a-zA-Z0-9-_]{1,100}"
	issueNumberPrefixRegexpTemplate = "(?P<issue_number_prefix>(https|http)://github\\.com/%s/%s/issues/|%s/%s#|#)"
	linkPrefixRegexpTemplate        = "(https|http)://github\\.com/(?P<org>%s)/(?P<repo>%s)/issues/"
	fullPrefixRegexpTemplate        = "(?P<org>%s)/(?P<repo>%s)#"
	shortPrefix                     = "#"

	associatePrefixGroupName   = "associate_prefix"
	issueNumberPrefixGroupName = "issue_number_prefix"
	issueNumberGroupName       = "issue_number"
	orgGroupName               = "org"
	repoGroupName              = "repo"
	defaultDelimiter           = ", "
)

var (
	issueNumberPrefixRegexp = fmt.Sprintf(issueNumberPrefixRegexpTemplate, orgRegexp, repoRegexp, orgRegexp, repoRegexp)
	linkPrefixRegexp        = fmt.Sprintf(linkPrefixRegexpTemplate, orgRegexp, repoRegexp)
	fullPrefixRegexp        = fmt.Sprintf(fullPrefixRegexpTemplate, orgRegexp, repoRegexp)
)

type issueNumberValue struct {
	associatePrefix string
	org             string
	repo            string
	number          int
}

type issueNumberData map[string]issueNumberValue

// put use map results to de duplicate data.
func (d issueNumberData) put(associatePrefix, org, repo string, issueNumber int) {
	key := fmt.Sprintf("%s-%s-%s-%d", associatePrefix, org, repo, issueNumber)
	d[key] = issueNumberValue{
		associatePrefix: associatePrefix,
		org:             org,
		repo:            repo,
		number:          issueNumber,
	}
}

// NormalizeIssueNumbers is an utils method in CommitTemplate that used to extract the issue numbers in the text
// and normalize it by a uniform format.
func NormalizeIssueNumbers(content, currOrg, currRepo, delimiter string) string {
	issueNumberBlockRegexp := fmt.Sprintf(issueNumberBlockRegexpTemplate, associatePrefixRegexp, issueNumberPrefixRegexp)
	compile, err := regexp.Compile(issueNumberBlockRegexp)
	if err != nil {
		panic(fmt.Errorf("failed to compile the normalize regexp: %v", err))
	}

	allMatches := compile.FindAllStringSubmatch(content, -1)
	groupNames := compile.SubexpNames()

	issueNumberMap := make(issueNumberData)
	for _, matches := range allMatches {
		associatePrefix := ""
		issueNumberPrefix := ""
		issueNumber := 0
		for i, groupName := range groupNames {
			switch groupName {
			case associatePrefixGroupName:
				associatePrefix = strings.ToLower(strings.TrimSpace(matches[i]))
			case issueNumberPrefixGroupName:
				issueNumberPrefix = strings.ToLower(strings.TrimSpace(matches[i]))
			case issueNumberGroupName:
				issueNumber, err = strconv.Atoi(strings.TrimSpace(matches[i]))
				if err != nil {
					panic(fmt.Errorf("failed to get issue number: %v", err))
				}
			}
		}

		if b, org, repo := isLinkPrefix(issueNumberPrefix); b {
			issueNumberMap.put(associatePrefix, org, repo, issueNumber)
		} else if b, org, repo := isFullPrefix(issueNumberPrefix); b {
			issueNumberMap.put(associatePrefix, org, repo, issueNumber)
		} else if isShortPrefix(issueNumberPrefix) {
			issueNumberMap.put(associatePrefix, currOrg, currRepo, issueNumber)
		}
	}

	// The issue number will be sorted in ascending order.
	issueNumberValues := make([]issueNumberValue, 0)
	for _, value := range issueNumberMap {
		issueNumberValues = append(issueNumberValues, value)
	}
	sort.Slice(issueNumberValues, func(i, j int) bool {
		return issueNumberValues[i].number < issueNumberValues[j].number
	})

	// Use a uniform prefix style.
	issueNumbers := make([]string, 0)
	for _, v := range issueNumberValues {
		issueNumbers = append(issueNumbers,
			shortenPrefix(v.associatePrefix, v.org, v.repo, currOrg, currRepo, v.number),
		)
	}

	result := ""
	if len(delimiter) == 0 {
		result = strings.Join(issueNumbers, defaultDelimiter)
	} else {
		result = strings.Join(issueNumbers, delimiter)
	}

	return result
}

// shortenPrefix used to simplify the prefix format. If it is the issue number of the same repository, the short prefix
// style will be used instead of the full prefix.
func shortenPrefix(associatePrefix, org, repo, currOrg, currRepo string, issueNumber int) string {
	if org == currOrg && repo == currRepo {
		return fmt.Sprintf("%s #%d", associatePrefix, issueNumber)
	}
	return fmt.Sprintf("%s %s/%s#%d", associatePrefix, org, repo, issueNumber)
}

// isLinkPrefix used to determine whether the prefix style of the issue number is link prefix,
// for example: https://github/com/pingcap/tidb/issues/123.
func isLinkPrefix(prefix string) (bool, string, string) {
	compile, err := regexp.Compile(linkPrefixRegexp)
	if err != nil {
		panic(fmt.Errorf("failed to compile the link prefix regexp: %v", err))
	}

	matches := compile.FindStringSubmatch(prefix)
	groupNames := compile.SubexpNames()

	if matches == nil {
		return false, "", ""
	}

	org := ""
	repo := ""
	for i, match := range matches {
		groupName := groupNames[i]
		if groupName == orgGroupName {
			org = match
		} else if groupName == repoGroupName {
			repo = match
		}
	}

	return true, org, repo
}

// isFullPrefix used to determine whether the prefix style of the issue number is full prefix,
// for example: pingcap/tidb#123.
func isFullPrefix(prefix string) (bool, string, string) {
	compile, err := regexp.Compile(fullPrefixRegexp)
	if err != nil {
		panic(fmt.Errorf("failed to compile the full prefix regexp: %v", err))
	}

	matches := compile.FindStringSubmatch(prefix)
	groupNames := compile.SubexpNames()

	if matches == nil {
		return false, "", ""
	}

	org := ""
	repo := ""
	for i, match := range matches {
		groupName := groupNames[i]
		if groupName == orgGroupName {
			org = match
		} else if groupName == repoGroupName {
			repo = match
		}
	}

	return true, org, repo
}

// isShortPrefix used to determine whether the prefix style of the issue number is short prefix, for example: #123.
func isShortPrefix(prefix string) bool {
	return prefix == shortPrefix
}
