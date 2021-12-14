# ti-community-lgtm

## Design Background

In the TiDB community, we use a multi-stage code review approach to collaboration. A PR will typically be reviewed by multiple people before it meets the basic criteria for merging. For example, when the PR is reviewed by the first person, the PR is labeled with `status/LGT1`. Then when the PR is reviewed by the second person, the PR is labeled with `status/LGT2`. Each SIG sets the default number of LGTMs required, which is usually 2.

ti-community-lgtm is a plugin that automatically adds LGTM labels to PRs based on commands and permissions. It is deployed as a standalone service, with Prow Hook forwarding GitHub webhook events to the plugin for processing.

## Permission design

The plugin is responsible for the collaborative process of code review with the following permissions:

- GitHub Approve
  - reviewers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
    - reviewers

- GitHub Request Changes
  - reviewers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
    - reviewers

## Design

The implementation of this plugin is based on how to integrate with GitHub's own review feature as a collaborative code review tool.

This feature is triggered in the following cases:

- Use Approve/Request Changes feature of GitHub

## Parameter Configuration

| Parameter Name               | Type     | Description                       |
|------------------------------|----------|-----------------------------------|
| repos                        | []string | Repositories                      |
| pull_owners_endpoint         | string   | PR owners RESTFUL API             |
| ignore_invalid_review_prompt | bool     | Do not prompt for invalid reviews |

For example:

```yml
ti-community-lgtm:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
      - tikv/pd
    pull_owners_endpoint: https://prow.tidb.io/ti-community-owners # You can define different URL to get owners
    ignore_invalid_review_prompt: true
```

## Reference Documents

- [command help](https://prow.tidb.io/command-help?repo=ti-community-infra%2Ftest-live#lgtm)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/lgtm)

## Q&A

### Why does `/lgtm [cancel]` take no effect anymore?

GitHub supports submitting review by Approve or Request Changes, which we always count as lgtm or reset lgtm status.

We don't want to support duplicate features that GitHub already provides, and regard the robot as an assistant, not a majordomo.

For original discussion, see also [#561](https://github.com/ti-community-infra/tichi/issues/561).

### Can I Approve my own PR?

No, you can't approve your own PR on GitHub.

### Why does Request Changes directly remove the results of my multiple reviews?

Because when a reviewer thinks that the code is faulty and needs to be re-reviewed, we think that the previous review is also faulty.

### Why do I have new commits LGTM related labels still kept?

This is because currently the TiDB community has a lot of code review phases, so if the bot cancels the LGTM as soon as a new commit is made, it can lead to a long PR review process and make PR merging difficult. So we loosened this part up to the reviewer. A reviewer can Request Changes to reset review status.
