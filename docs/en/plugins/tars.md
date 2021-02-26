# ti-community-tars

## Design Background

Since most CI systems are tested on the current PR, the following problems occur when the Base of our PR falls behind:

- PR1: Rename bifurcate() to bifurcateCrab()
- PR2: Call bifurcate()

In this case, both PRs will be tested with the current master as the base branch, and both PRs will pass. However, once PR1 is merged into the master branch first, and the second PR is merged (because the test also passes), it causes a master error that `bifurcate` is not found.

To solve this problem GitHub provides a branch protection option called `Require branches to be up to date before merging`. When this option is turned on, PR will only merge when the latest Base branch is in use. **This solves the problem, but it requires you to manually click the GitHub button to merge the latest Base branch into the PR, which is a mechanical and repetitive task**.

ti-community-tars is designed to solve this problem by automatically detecting if a PR is out of date when there is a reply or other update to the PR, and then helping us automatically merge the latest Base branch into the PR. In addition, it also supports periodically scanning all PRs from all repositories where the plugin is configured and trying to select the first created PR to be updated. (Since all PR merges require the latest Base, it would be a waste of testing resources if they were all updated)

## Design

To implement this plugin, we not only need to automatically detect and update PRs when they are updated or have replies. We also need to take into account that some PRs will not be updated or responded to after the code review is over, and these PRs may not be merged due to the `Require branches to be up to date before merging` option. So the plugin also needs to support regular scanning to update these PRs.

In addition, most PRs that do not meet the merge criteria may not want to be automatically updated. Because after the automatic update, we need to pull the latest update when we have a new commit push locally. So we specify which PRs need to be updated via the label configuration item.

## Parameter Configuration 

| Parameter Name  | Type     | Description                                    |
| --------------- | -------- | ---------------------------------------------- |
| repos           | []string | Repositories                                   |
| message         | string   | Messages replied to after the automatic update |
| only_when_label | string   | Only help update when PR adds the label        |

For example:

```yaml
ti-community-tars:
  - repos:
      - ti-community-infra/test-dev
    only_when_label: "status/can-merge"
    message: "Your PR was out of date, I have automatically updated it for you."
```

## Reference Documents

- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/tars)

## Q&A

### How often are scans performed?

It is currently twenty minutes, and will be adjusted later according to the number of repositories used.