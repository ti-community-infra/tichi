# wip(Work In Process)

## Design Background

When we submit a PR, we may need to commit multiple changes to complete the PR for some of the more complex fixes, and we would prefer that reviewers not come in and do Code Review while we are still in the process of modifying the PR.

[wip](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/wip) adds or removes `do-not-merge/work-in-progress` label, and with [tide](components/tide.md) to prevent merging of PRs.

## Design

This plugin was designed and developed by the Kubernetes community and is very simple to implement. It automatically adds or removes the `do-not-merge/work-in-progress` label when our PR is in draft status or when the PR title contains `WIP`.

## Parameter Configuration 

No configuration

## Reference documents

- [wip doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/wip)

## Q&A

### Will it automatically add and remove `do-not-merge/work-in-progress` label as the status or title of my PR changes?

Yes, it will automatically remove the label when your PR is not draft status or the title does not contain `WIP`.
