# ti-community-lgtm

## Design Background

In the TiDB community, we use a multi-stage code review approach to collaboration. A PR will typically be reviewed by multiple people before it meets the basic criteria for merging. For example, when the PR is reviewed by the first person, the PR is labeled with `status/LGT1`. Then when the PR is reviewed by the second person, the PR is labeled with `status/LGT2`. Each SIG sets the default number of LGTMs required, which is usually 2.

ti-community-lgtm is a plugin that automatically adds LGTM labels to PRs based on commands and permissions. It is deployed as a standalone service, with Prow Hook forwarding GitHub webhook events to the plugin for processing.

## Permission design

The plugin is responsible for the collaborative process of code review with the following permissions:

- `/lgtm` Or GitHub Approve
  - reviewers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
    - reviewers

- `/lgtm cancel` Or GitHub Request Changes
  - reviewers
    - maintainers
    - techLeaders
    - coLeaders
    - committers
    - reviewers
  - **PR author**


## Design

The implementation of this plugin was not only about its support for the `/lgtm` comment command, but also about how it would integrate with GitHub's own review functionality as a collaborative code review tool.

This feature is triggered in the following cases(**Commands are not case sensitive**):

- Use `/lgtm [cancel]` in comment
- Use GitHub's own Approve/Request Changes feature(if the review_acts_as_lgtm option is turned on)

**Special attention**:

- The command must start with `/` (**this is the basic specification for all commands**)

## Parameter Configuration 

| Parameter Name       | Type     | Description                                                                 |
| -------------------- | -------- | --------------------------------------------------------------------------- |
| repos                | []string | Repositories                                                                |
| review_acts_as_lgtm  | bool     | Whether to treat GitHub Approve/Request Changes as a valid `/lgtm [cancel]` |
| pull_owners_endpoint | string   | PR owners RESTFUL API                                                       |

For example:

```yml
ti-community-lgtm:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
      - tikv/pd
    review_acts_as_lgtm: true
    pull_owners_endpoint: https://prow.tidb.io/ti-community-owners # You can define different URL to get owners
```

## Reference Documents

- [command help](https://prow.tidb.io/command-help?repo=ti-community-infra%2Ftest-live#lgtm)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/lgtm)

## Q&A

### Can I `/lgtm` my own PR?

No, even if you have reviewer access, it won't be credited as a valid code review, just like you can't approve your own PR on GitHub.

### Why does `/lgtm cancel` directly remove the results of my multiple reviews?

Because when a reviewer thinks that the code is faulty and needs to be re-reviewed, we think that the previous review is also faulty.

### Why do I have new commits LGTM related labels still kept?

This is because currently the TiDB community has a lot of code review phases, so if the bot cancels the LGTM as soon as a new commit is made, it can lead to a long PR review process and make PR merging difficult. So we loosened this part up to the reviewer and the author, and you can `/lgtm cancel` yourself when you feel you need to review again.
