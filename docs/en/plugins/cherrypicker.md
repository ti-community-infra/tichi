# ti-community-cherrypicker

## Design Background

In the TiDB community, there are several large repositories with multiple branches being maintained. When we make changes to the master's code and created the PR, those changes may need to be applied to other branches as well. Relying on manual cherry-picking can be a huge workload and error-prone.

ti-community-cherrypicker will help us automatically cherry-pick PR changes to another branch and create PRs automatically. **It also supports a 3-way merge to force cherry-pick code into the Base branch in case of code conflicts.**

## Permission design

This plugin is primarily responsible for cherry-picking code and creating PRs with the following permissions:

- `allow_all` set to true, all GitHub users can trigger `/cherry-pick some-branch`
- `allow_all` set to false, only members of the repo's Org can trigger `/cherry-pick some-branch`

## Design

The implementation of this plugin considers the following two main cases of PR:

- PR Conflict-free
  - We can directly download [the patch file provided by GitHub](https://stackoverflow.com/questions/6188591/download-github-pull-request-as-unified-diff) for a 3-way mode [git am](https://git-scm.com/docs/git-am) operation.
- PR has conflicts
  - We can't apply the patch directly because the commits in the patch will be applied one by one, and this process may result in multiple conflicts, and the process of resolving conflicts will be very complicated.
  - We can just cherry-pick the merge_commit_sha that is generated on GitHub when the current PR merged (rebase/merge/squash merges all generate this commit), so we can cherry-pick the entire PR to the Base branch at once and only need to resolve the conflict once.

Note: **The above `resolve conflict` means that the tool will `git add` the conflicting code directly and commit it to a new PR, not actually modify the code to resolve the conflict**.

In addition to implementing the core functionality of cherry-pick, it also supports a number of other features:

- Use labels to mark which branches needs cherry-pick
- Assign the PR of cherry-pick to the author or requester (the person who requested cherry-pick)
- Copy the labels already added for the current PR

## Parameter Configuration 

| Parameter Name           | Type     | Description                                                                                                                                      |
| ------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| repos                    | []string | Repositories                                                                                                                                     |
| allow_all                | bool     | Whether to allow non-Org members to trigger cherry-pick                                                                                          |
| create_issue_on_conflict | bool     | Whether to create an Issue to track when there is a code conflict, if false then the conflicting code will be committed to the new PR by default |
| label_prefix             | string   | The prefix of the label that triggers cherry-pick, default is `cherrypick/`                                                                      |
| picked_label_prefix      | string   | The label prefix of the PR created by cherry-pick (e.g. `type/cherry-pick-for-release-5.0`)                                                      |
| exclude_labels           | []string | Some labels that you don't want to be automatically copied by the plugin (e.g. some labels that control code merging)                            |

For example:

```yml
ti-community-cherrypicker:
  - repos:
      - pingcap/dumpling
    label_prefix: needs-cherry-pick-
    allow_all: true
    create_issue_on_conflict: false
    excludeLabels:
      - status/can-merge
      - status/LGT1
      - status/LGT2
      - status/LGT3
```

## Reference Documents

- [command help](https://prow.tidb.io/command-help?repo=ti-community-infra%2Ftest-live#merge)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/merge)

## Q&A

### If there is a conflict between the PR of the robot cherry-pick and the target branch code, how can I change that PR?

**If you have write access to the repository, you can modify the PR directly.** 

GitHub supports [maintainers directly modifying code in fork repositories](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/allowing-changes-to-a-pull-request-branch-created-from-a-fork). This option is turned on by default when the robot creates a PR for the maintainer to make changes to the PR.

### How do I checkout a robot's PR to make changes?

GitHub has recommended in the PR page that you can use the official GitHub [cli](https://github.com/cli/cli) to checkout and make changes. **See the `Open with` dropdown in the upper right corner of the PR page for more information.**

### Why is the bot dropping some of my commits?

This only happens when you make code changes in the merge Base commit, because GitHub automatically deletes the merge Base commit when it generates the patch.

For example:
> 
> There are 8 commits in this [PR](https://github.com/pingcap/dm/pull/1638), but its [patch](https://patch-diff.githubusercontent.com/raw/pingcap/dm/pull/1638.patch) has only 5 commits. **Because GitHub automatically removes the merge master commits.**
> 
> You can see that cherry-pick's [PR](https://github.com/pingcap/dm/pull/1650) also has only 5 commits, and because this [commit](https://github.com/pingcap/dm/pull/1638/commits/8c08720653a6904a029e76bd66d499ef73c385fc) in the original PR not only merged the master but also made changes to the code, the commit was eventually lost.

**So it is recommended that you do not modify the code in the merge Base commit, as this will result in code loss.**

