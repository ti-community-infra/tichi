# ti-community-contribution

## Design Background

In the TiDB community, there will be a large number of external contributors participating in the community to contribute. We need to distinguish between internal and external contributors' PRs, and prioritize helping to review external contributors' PRs, especially for first-time PRs, so that external contributors can have a good contribution experience.

ti-community-contribution will help us distinguish between the PRs of internal and external contributors by adding the label `contribution` or `first-time-contributor` to the PRs of external contributors.

## Design

This plugin adds a `contribution` label to a PR based on whether the author is a member of the Org where the repository is located, and also adds a `first-time-contributor` tag to a PR if it is the first time the author has submitted a PR to the repository or the first time the PR has been submitted on GitHub.

## Parameter Configuration 

| Parameter Name | Type     | Description                            |
| -------------- | -------- | -------------------------------------- |
| repos          | []string | Repositories                           |
| message        | string   | Message replied to after adding labels |


For example:

```yml
ti-community-merge:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
      - tikv/pd
    message: "Thank you for your contribution, we have some references for you."
```

## 参考文档

- [RFC](https://github.com/ti-community-infra/rfcs/blob/main/active-rfcs/0001-contribution.md)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/contribution)

## Q&A

### Why is the distinction between external contributions based on whether or not they are Org members?

Because if a member of Org means that he is at least a reviewer or above (**only reviewers or above will be invited to Org**), he is already familiar with the PR process and does not need much help.