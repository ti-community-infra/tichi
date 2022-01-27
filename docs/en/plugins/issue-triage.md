# ti-community-issue-triage

## Design Background

In the existing versioning model of the TiDB community, TiDB maintains multiple releases at the same time, and bug issues of critical or major severity need to be fixed on the affected release branch that is still being maintained.

For bug issues, we will label the affected release branches with `affects/*` labels, e.g. `affects/5.1` label means the bug affects the releases under `release-5.1` branch, but there are some problems if an issue is not labeled with `affects/ *` tag, it is difficult to determine whether the issue has been diagnosed or does not affect the release branch. In order to prevent bug issue fixes from being missed, we have developed [a new Triage workflow](https://github.com/pingcap/community/blob/master/votes/0625-new-triage-method-for- cherrypick.md).

In the new process, before merging pull requests for fixing critical or major severity bug issues, the triage process of the bug issue needs to be completed to identify all release branches affected by the bug issue, and when the bug issue triage is completed, the bot will When the bug issue triage is completed, the bot will automatically convert the `affects-x.y` label to the corresponding `needs-cherry-pick-release-x.y` label and add it to the pull request, and when the PR meets other conditions to complete the merge, then the bot will automatically create the fix PR cherry pick to each affected release branch. The

The ti-community-issue-triage plugin is designed to manage and automate this process.

## Design

### Issue side

For critical or major severity bug issues.

- When a new bug issue is labeled with `severity/critical` or `severity/major`, the bot will add the corresponding `may-affects-x.y` label to the issue based on the list of releases currently being maintained.

- When the issue passes diagnostics and it is confirmed that the issue affects the `release-x.y` branch, contributors can use the `/affects x.y` command to add the corresponding `affects-x.y` label, and the bot will remove the corresponding `may-affects-x.y` label at the same time.

- When the `may-affects-x.y` label on an issue changes, it will rerun all pull requests opened on the **default branch** (e.g. `master` branch) associated with it

### Pull request side

For pull requests that fix related bug issues and linked with them, we add a check named `check-issue-triage-complete` that will ensure that the bug issues are triaged before the pull request is merged.

By default, the plugin will trigger the check at the right time, and if it doesn't, contributors can trigger it manually with the `/run-check-issue-triage-complete` command.

The plugin determines whether a pull request is triaged according to the following rules.

- PR-linked issues are determined by the `Issue Number: ` line

- PR-linked issues must contain the `type/*` label

- PR-linked bug issues must contain the `severity/*` label

- A bug issue linked with a PR labeled with `severity/critical` or `severity/major` cannot contain any `may-affects-x.y` labels, if it does, it is considered to have not completed triage and does not satisfy the merge condition

- If PR linked with multiple bug issues, then all of them need to satisfy the above condition

If `check-issue-triage-complete` does not pass, the bot will automatically label `do-not-merge/needs-triage-completed` to prevent the PR from merging.

When the `check-issue-triage-complete` check passes, the bot removes the `do-not-merge/needs-triage-completed` label and automatically labels the PR with `needs- cherry-pick-release-x.y` labels for all associated bug issues.

## Parameter Configuration

| parameter name                  | type     | description                                                                         |
|---------------------------------|----------|-------------------------------------------------------------------------------------|
| repos                           | []string | Configuration effective repository                                                  |
| maintain_versions               | []string | The version number of the release branch being maintained by the repository         |
| affects_label_prefix            | string   | The label prefix that identifies the release branch affected by the issue           |
| may_affects_label_prefix        | string   | The label prefix that identifies the release branch that the issue may affect       |
| linked_issue_needs_triage_label | string   | The label that identifies the PR's associated issue that requires triage completion |
| need_cherry_pick_label_prefix   | string   | The prefix identifying the PR's need for cherry-pick to the release branch          |
| status_target_url               | string   | The details URL of the status check                                                 |

Example:

```yml
ti-community-issue-triage:
  - repos:
      - ti-community-infra/test-dev
    maintain_versions:
      - "5.1"
      - "5.2"
      - "5.3"
    affects_label_prefix: "affects/"
    may_affects_label_prefix: "may-affects/"
    linked_issue_needs_triage_label: "do-not-merge/needs-triage_completed"
    need_cherry_pick_label_prefix: "needs-cherry-pick-release-"
    status_target_url: "https://book.prow.tidb.io/#/plugins/issue-triage"
```

## Reference Documents

- [0625-new-triage-method-for-cherrypick.md](https://github.com/pingcap/community/blob/master/votes/0625-new-triage-method-for- cherrypick.md)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/issuetriage)

