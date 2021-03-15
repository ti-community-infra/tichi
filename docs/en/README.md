# Prow Introduction

## About Prow

[Prow](https://github.com/kubernetes/test-infra/tree/master/prow) is a Kubernetes based CI/CD system. 
Jobs can be triggered by various types of events and report their status to many services. In addition to job execution, Prow provides GitHub automation in the form of policy enforcement, chat-ops via /foo style commands, and automatic PR merging.

## Prow in TiDB community

In [tichi](https://github.com/ti-community-infra/tichi), the focus is on automating and standardizing the [TiDB](https://github.com/pingcap/tidb) community's collaboration process using the GitHub automation features provided by Prow.

Because Prow is so [extensible](https://github.com/kubernetes/test-infra/tree/master/prow/plugins), it is possible to write and customize [plugins](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins) according to your own community practice specifications.
The TiDB community has customized a number of its own plugins based on this feature.

## Common plugin list

The following shows some of the more commonly used components or plugins in the TiDB community. The external plugins beginning with `ti-community-` are those developed and maintained by ti-community-infra SIG.

| plugin name                | plugin type     | introduction                                                                                                                                   |
| -------------------------- | --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| tide                       | basic component | Manage the GitHub PR pool through some given conditions and automatically merge PR that meets the conditions.                                  |
| rerere                     | basic component | Push the code to a branch dedicated to re-testing for re-testing.                                                                              |
| ti-community-owners        | external plugin | Determine the reviewer and committer of the PR based on information such as SIG or Github permissions.                                         |
| ti-community-lgtm          | external plugin | Add command to add or cancel the `status/LGT*` label.                                                                                          |
| ti-community-merge         | external plugin | Add or cancel the PR's `can-merge` status via commands.                                                                                        |
| ti-community-blunderbuss   | external plugin | Mainly responsible for automatically assigning reviewers based on SIG or Github permissions.                                                   |
| ti-community-autoresponder | external plugin | Automatically reply based on the content of the comment.                                                                                       |
| ti-community-tars          | external plugin | Mainly responsible for automatically merging the main branch into the current PR to ensure that the current PR's Base is kept up to date.      |
| ti-community-label         | external plugin | Add labels to PR or Issue via commands.                                                                                                        |
| ti-community-label-blocker | external plugin | Mainly responsible for preventing users from illegal operations on certain sensitive labels.                                                   |
| need-rebase                | external plugin | When the PR needs to rebase, add labels or add comments to remind the PR author to rebase.                                                     |
| require-matching-label     | external plugin | When a PR or Issue lacks a relevant label, add a label or comment to remind contributors to supplement.                                        |
| hold                       | external plugin | Add or cancel the non-combinable status of PR through the `/[un]hold` command.                                                                 |
| assign                     | external plugin | Add or cancel the assignee of PR or Issue through the `/[un]assign` command.                                                                   |
| size                       | external plugin | Evaluate the size of the PR based on the number of lines of code modification, and label the PR with `size/*`.                                 |
| lifecycle                  | external plugin | Use labels to mark the life cycle of Issue or PR.                                                                                              |
| wip                        | external plugin | Mark the PR that is still under development as `work-in-process` status, and prevent the automatic allocation of reviewer and PR from merging. |
| welcome                    | external plugin | Send a welcome message to contributors who have contributed for the first time through a robot.                                                |
| label_sync                 | tool            | Able to synchronize the labels configured in the yaml file to one or more repositories.                                                        |



At the same time, you can find all currently available components or plugins in [Plugins](https://prow.tidb.io/plugins) page, or in [Command](https://prow.tidb.io/command-help) to view the commands available in the specified repository.

If you want to implement a new functional module through tichi, you can propose it through [RFC](https://github.com/ti-community-infra/rfcs) so that we can communicate widely about it in the community. So as to finally determine the specific requirements of the new function.

## About the book

The purpose of this book is to serve as a handbook for collaborators in the TiDB community, and as an example of how to use Prow for the rest of the community.

