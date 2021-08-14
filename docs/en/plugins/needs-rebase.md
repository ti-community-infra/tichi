# needs-rebase

## Design Background

GitHub doesn't usually alert PR authors to resolve PR conflicts, which can result in our PR not being automatically merged by the bot and requiring to alert them manually to resolve the conflict. Also, we can't see which one have conflicts in the PRs list.

[needs-rebase](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins/needs-rebase) can periodically check for conflicts in PR by adding the `needs-rebase` label and remind PR authors to resolve conflicts.

## Design

This plugin was designed and developed by the Kubernetes community. They implemented it with the idea that not only should all PRs be scanned periodically to add or remove `needs-rebase` label, but also that those PRs that are active and have replies should have `needs-rebase` label added as soon as possible to remind them to resolve conflicts.

## Parameter Configuration

No configuration

## Reference documents

- [command help](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins/needs-rebase)

## Q&A

### Is it disturbing to PR authors that the bot comments while adding `needs-rebase` label?

> https://github.com/ti-community-infra/tichi/issues/408

This is because GitHub doesn't alert you when a PR conflict has occurred, and GitHub doesn't notify you when a bot adds `needs-rebase`, so we have to explicitly notify you via a reply.
Also, **after you resolve the conflict the bot automatically removes the `needs-rebase` label and deletes the outdated and useless replies. **

### How often are scans performed?

The automatic scan is performed once every 24 hours.