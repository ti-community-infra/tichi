package externalplugins

import (
	"strings"
	"testing"

	"k8s.io/test-infra/prow/github"
)

func TestFormatICResponse(t *testing.T) {
	ic := github.IssueComment{
		Body:    "Looks neat.\r\nI like it.\r\n",
		User:    github.User{Login: "ca"},
		HTMLURL: "happygoodsite.com",
	}
	s := "you are a nice person."
	out := FormatICResponse(ic, s)
	if !strings.HasPrefix(out, "@ca: you are a nice person.") {
		t.Errorf("Expected compliments to the comment author, got:\n%s", out)
	}
	if !strings.Contains(out, ">I like it.\r\n") {
		t.Errorf("Expected quotes, got:\n%s", out)
	}
}

func TestFormatResponseRaw(t *testing.T) {
	body := "Looks neat.\r\nI like it.\r\n"
	user := "ca"
	htmlURL := "happygoodsite.com"
	comment := "you are a nice person."

	out := FormatResponseRaw(body, htmlURL, user, comment)
	if !strings.HasPrefix(out, "@ca: you are a nice person.") {
		t.Errorf("Expected compliments to the comment author, got:\n%s", out)
	}
	if !strings.Contains(out, ">I like it.\r\n") {
		t.Errorf("Expected quotes, got:\n%s", out)
	}
}
