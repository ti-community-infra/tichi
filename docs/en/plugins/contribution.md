# ti-community-contribution

## Design Background

We would like to give extra visibility to pull request created by contributors not a member of the org where the repository is located, in order to search and deal with these contributions.

ti-community-contribution will help us distinguish whether the author of pull request is a member of the org by adding the label `contribution` or `first-time-contributor`.

## Design

This plugin adds a `contribution` label to a PR based on whether the author is a member of the org where the repository is located, and also adds a `first-time-contributor` label to a PR if it is the first time the author has submitted a PR to the repository or the first time the PR has been submitted on GitHub.

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

## Reference Documents

- [RFC](https://github.com/ti-community-infra/rfcs/blob/main/active-rfcs/0001-contribution.md)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/contribution)
