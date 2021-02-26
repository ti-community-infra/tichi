# ti-community-blunderbuss

## Design Background

In the TiDB community, because a PR goes through multiple stages of review, we want to be able to automatically assign reviewers when a PR is created.

ti-community-blunderbuss is responsible for automatically assigning reviewers based on the permissions assigned by ti-community-owners when a PR is created, but in addition, we need to consider the case where we need to request another review if a reviewer has been unresponsive for a long time So we support the `/auto-cc` command to trigger the reassignment of reviewers.

In fact, in some TiDB community repositories, most PRs require sig tags, and reviewers can only be automatically assigned after sig tags are added, so we need to restrict this through configuration to reduce unnecessary auto-assignment.

## Permission design

This plugin is mainly responsible for the automatic assignment of reviewers, so we set the permissions to allow all GitHub users to use this feature.

## Design

This plugin is mainly based on the blunderbuss plugin for Kubernetes. Based on it, we rely on ti-community-owners to automate the assignment of reviewers for PR.

The allocation strategy is:

- If the number of reviewers with permission is less than or equal to `max_request_count`
  - Assign all reviewers with permissions
- If the number of reviewers with permission is greater than `max_request_count`
  - Get all the file changes of PR, find out the historical contributors of these changed files, and calculate the weights based on the number of changes made by the contributors to the files for weighted random assignment

If a repository requires a PR with a sig label for auto-assignment, then creating the PR, using the `/auto-cc` command will not auto-assign until the PR is labeled with the sig-related label. The plugin will only automatically assign reviewers after we add the sig labels.

**Special note**: When the `/cc` command is used in the body of a PR or reviewers have been manually specified, the plugin will not automatically assign them. However, there is no such restriction with the `/auto-cc` command.

## Parameter Configuration 

| Parameter Name        | Type     | Description                                                                                         |
| --------------------- | -------- | --------------------------------------------------------------------------------------------------- |
| repos                 | []string | Repositories                                                                                        |
| pull_owners_endpoint  | string   | PR owners RESTFUL API address                                                                       |
| max_request_count     | int      | Maximum number of assignees (not configured to assign all reviewers)                                |
| exclude_reviewers     | []string | Reviewers who do not participate in auto-assignment (for some reviewers who may be inactive)        |
| grace_period_duration | int      | Configure the waiting time in seconds for other plugins to add sig labels, the default is 5 seconds |
| require_sig_label     | bool     | Whether the PR must have a SIG label to allow automatic assignment of reviewers                     |

For example:

```yml
ti-community-blunderbuss:
  - repos:
      - ti-community-infra/test-live
    pull_owners_endpoint: https://prow-dev.tidb.io/ti-community-owners
    max_request_count: 1
    exclude_reviewers:
      # Bots
      - ti-chi-bot
      - rustin-bot
      # Inactive reviewers
      - sykp241095
      - AndreMouche
    grace_period_duration: 5
    require_sig_label: true
```

## Reference Documents

- [command help](https://prow.tidb.io/command-help?repo=ti-community-infra%2Fconfigs#auto_cc)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/blunderbuss)

## Q&A

### Why doesn't the /auto-cc command automatically assign reviewers?

It may be because your repository has `require_sig_label` set, which causes reviewers to not be automatically assigned until the sig label is labeled.


