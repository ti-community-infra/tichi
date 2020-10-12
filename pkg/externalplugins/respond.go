package externalplugins

import (
	"fmt"
	"strings"

	"k8s.io/test-infra/prow/github"
)

// AboutThisBotWithoutCommands contains the message that explains how to interact with the bot.
const AboutThisBotWithoutCommands = "Instructions for interacting with me using PR comments are available " +
	"[here](https://github.com/pingcap/community/blob/master/contributors/command-help.md).  " +
	"If you have questions or suggestions related to my behavior, " +
	"please file an issue against the " +
	"[tidb-community-bots/prow-config]" +
	"(https://github.com/tidb-community-bots/prow-config/issues/new?title=Prow%20issue:) repository."

// AboutThisBotCommands contains the message that links to the commands the bot understand.
const AboutThisBotCommands = "I understand the commands that are listed [here](https://go.k8s.io/bot-commands)."

// AboutThisBot contains the text of both AboutThisBotWithoutCommands and AboutThisBotCommands.
const AboutThisBot = AboutThisBotWithoutCommands + " " + AboutThisBotCommands

// FormatResponse nicely formats a response to a generic reason.
func FormatResponse(to, message, reason string) string {
	format := `@%s: %s

<details>

%s

%s
</details>`

	return fmt.Sprintf(format, to, message, reason, AboutThisBotWithoutCommands)
}

// FormatSimpleResponse formats a response that does not warrant additional explanation in the
// details section.
func FormatSimpleResponse(to, message string) string {
	format := `@%s: %s

<details>

%s
</details>`

	return fmt.Sprintf(format, to, message, AboutThisBotWithoutCommands)
}

// FormatICResponse nicely formats a response to an issue comment.
func FormatICResponse(ic github.IssueComment, s string) string {
	return FormatResponseRaw(ic.Body, ic.HTMLURL, ic.User.Login, s)
}

// FormatResponseRaw nicely formats a response for one does not have an issue comment
func FormatResponseRaw(body, bodyURL, login, reply string) string {
	format := `In response to [this](%s):

%s
`
	// Quote the user's comment by prepending ">" to each line.
	var quoted []string
	for _, l := range strings.Split(body, "\n") {
		quoted = append(quoted, ">"+l)
	}
	return FormatResponse(login, reply, fmt.Sprintf(format, bodyURL, strings.Join(quoted, "\n")))
}
