# hold

## Design Background

During our Code Review process, there may be a situation where a PR change is fine, but the PR has significant side effects that require others to be involved in evaluating whether and when it can be merged.

[hold](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/hold) adds or removes the `do-not-merge/hold` label for PRs by using the command `/hold [cancel]` and with [tide](components/tide.md) prevents merging of PRs.

## Design

Designed and developed by the Kubernetes community, this plugin is a simple implementation that controls the merging of PRs by adding or removing the `do-not-merge/hold` label via `/hold [cancel]`.

## Parameter Configuration

No configuration

## Reference documents

- [command-help](https://prow.tidb.net/command-help#hold)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/hold)

## Q&A

### When should I use this feature?

The code is fine and you feel comfortable agreeing to the changes, but the changes may have some side effects and more people need to carefully evaluate whether or when the changes can be merged.
