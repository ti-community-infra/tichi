# Prow Introduction

## About Prow

[Prow](https://github.com/kubernetes/test-infra/tree/master/prow) is a Kubernetes based CI/CD system. 
Jobs can be triggered by various types of events and report their status to many services. In addition to job execution, Prow provides GitHub automation in the form of policy enforcement, chat-ops via /foo style commands, and automatic PR merging.

## Prow in TiDB community

In [tichi](https://github.com/ti-community-infra/tichi), the focus is on automating and standardizing the [TiDB](https://github.com/pingcap/tidb) community's collaboration process using the GitHub automation features provided by Prow.

Because Prow is so [extensible](https://github.com/kubernetes/test-infra/tree/master/prow/plugins), it is possible to write and customize [plugins](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins) according to your own community practice specifications.
The TiDB community has customized a number of its own plugins based on this feature.

## Common plugin list

The following shows some commonly used components or plugins in the TiDB community. The external plugins whose name begin with `ti-community-` are developed and maintained by the [Community Infra SIG](https://developer.tidb.io/SIG/community-infra) of TiDB community. If you encounter any problems during use, you can contact us in [Slack](https://slack.tidb.io/invite?team=tidb-community&channel=sig-community-infra&ref=github).

| plugin name                     | plugin type     | introduction                                                                                                                              |
| ------------------------------- | --------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| tide                            | basic component | Manage the GitHub PR pool through some given conditions and automatically merge PR that meets the conditions.                             |
| ti-community-owners             | external plugin | Determine the reviewer and committer of the PR based on information such as SIG or Github permissions.                                    |
| ti-community-lgtm               | external plugin | Add, update or delete the `status/LGT*` label.                                                                                     |
| ti-community-merge              | external plugin | Add or delete the `status/can-merge` label for PR by command.                                                                             |
| ti-community-blunderbuss        | external plugin | Mainly responsible for automatically assigning reviewers based on SIG or Github permissions.                                              |
| ti-community-autoresponder      | external plugin | Automatically reply based on the content of the comment.                                                                                  |
| ti-community-tars               | external plugin | Mainly responsible for automatically merging the main branch into the current PR to ensure that the current PR's Base is kept up to date. |
| ti-community-label              | external plugin | Add labels to PR or Issue via commands.                                                                                                   |
| ti-community-label-blocker      | external plugin | Mainly responsible for preventing users from illegal operations on certain sensitive labels.                                              |
| ti-community-contribution       | external plugin | Mainly responsible for adding `contribution` or `first-time-contributor` labels to the PRs of external contributors.                      |
| ti-community-label-cherrypicker | external plugin | Mainly responsible for cherry-pick PR to other target branches.                                                                           |
| needs-rebase                    | external plugin | When the PR needs to rebase, add labels or add comments to remind the PR author to rebase.                                                |
| require-matching-label          | internal plugin | When a PR or Issue lacks a relevant label, add a label or comment to remind contributors to supplement.                                   |
| hold                            | internal plugin | Add or cancel the non-combinable status of PR through the `/[un]hold` command.                                                            |
| assign                          | internal plugin | Add or cancel the assignee of PR or Issue through the `/[un]assign` command.                                                              |
| size                            | internal plugin | Evaluate the size of the PR based on the number of lines of code modification, and label the PR with `size/*`.                            |
| lifecycle                       | internal plugin | Use labels to mark the life cycle of Issue or PR.                                                                                         |
| wip                             | internal plugin | Add the `do-not-merge/work-in-progress` label to the PR under development to prevent the automatic assignment of reviewer and PR merge.   |
| welcome                         | internal plugin | Send a welcome message to contributors who have contributed for the first time through a robot.                                           |
| release-note                    | internal plugin | It is mainly responsible for detecting whether a PR has added a release note.                                                             |
| label_sync                      | tool            | Able to synchronize the labels configured in the yaml file to one or more repositories.                                                   |
| autobump                        | tool            | Update the version of upstream Prow and its related components and plugins by automatically submitting Pull Requests.                     |

At the same time, you can find all currently available components or plugins in [Plugins](https://prow.tidb.io/plugins) page, or in [Command](https://prow.tidb.io/command-help) to view the commands available in the specified repository.

If you want to implement a new feature through tichi, you can put forward your requirements through [RFC](https://github.com/ti-community-infra/rfcs) so that we can communicate widely in the community. So as to finally determine the specific requirements and implementation plan of the new feature.

## About the book

The purpose of this book is to serve as a handbook for collaborators in the TiDB community, and as an example of how to use Prow for the rest of the community.

