# ti-community-tars

## Design Background

Since most CI systems are tested on the current Base of the PR, the following problems occur when the Base of our PR falls behind:

- PR1: Rename bifurcate() to bifurcateCrab()
- PR2: Call bifurcate()

In this case, both PRs will be tested with the current master as the base branch, and both PRs will pass. However, once PR1 is merged into the master branch first, and the second PR is merged (because the test also passes), it causes a master error that `bifurcate` is not found.

To solve this problem GitHub provides a branch protection option called `Require branches to be up to date before merging`. When this option is turned on, PR will only merge when the latest Base branch is in use. **This solves the problem, but it requires you to manually click the GitHub button to merge the latest Base branch into the PR, which is a mechanical and repetitive task**.

ti-community-tars is designed to solve this problem by helping us automatically merge the latest Base branch into the PR when the PR is commented to, updated, or when a new commit is made to the Base branch. In addition to this, it also supports periodic scanning of all PRs for all repositories where the plugin is configured and updating them **one by one**.

## Design

The following scenarios need to be considered to implement the plugin:
- PR with reply or update
  - When there is a reply or update to the PR, it means that someone is paying attention to the PR and may want the PR to be merged as soon as possible, so we should respond and update the PR as soon as possible
- Base branch has new commits
  - As soon as a new commit is made to the Base branch, we should look for other PRs that can be merged and update the latest Base to the PR
  - We can't update all PRs at once because we can only merge at most one PR at a time, so we should choose the PRs that were created the earliest and can be merged
- Regular scans and updates
  - Since the option mentioned above is turned on to ensure that the PRs pass the test even after merging the latest Base, we also need to periodically merge the latest Base into these PRs to test and resolve possible problems as soon as possible. This periodically updates these PRs one by one, and also prevents the merge queue from being blocked due to the failure of the previous PR tests

In addition, most PRs that do not meet the merge criteria do not want to be automatically updated. Because after the automatic update, we need to pull the latest update when we have a new commit push locally. So we specify which PRs need to be updated via the label configuration item(default is `status/can-merge`).

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

### How often will regular scans be performed?

This is currently 20 minutes, updating the earliest created PR for each repository's different branches each time.