# Black house plugin

## Design Background

Flaky test will affect the success rate and stability of job builds, and will cause CI resource inefficient.
We need a technical mechanism governance flaky tests in force or encourage mode.

## Permission design

## Design

- On pull request block label status: [Status graph](../../UML/flaky/pr-label-status.puml)
- Pull request event flow: [PR event flow](../../UML/flaky/pr-flow.puml)
- Issue event flow: [Issue event flow](../../UML/flaky/issue-flow.puml)


## Parameter Configuration 

For example:

```yml
ti-community-flaky:
  - repos:
      - <org1>
      - <org2/repo2>
    dept_label: flaky-debt # label to identify debt repaying kind PR or debt issues.
    dept_duration_threshold: 168h # 1 week
    dept_issue_pr_rate: 0 # how many overed threshold flaky issues will block one opened PR, value <= 0 will block all opened PRs.
    block_label: do-not-merge/flaky-debt-not-repaid # label to block PR to be merged.
    comment_enabled: true # when enable, it will cost github api request quota.
```