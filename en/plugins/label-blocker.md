# ti-community-label-blocker

## Design Background

Throughout the [PR collaboration process](en/workflows/pr.md), some labels are recognized and used by the bot, for example, to determine if a PR can be merged based on whether it has a `status/can-merge` label. For these sensitive labels, we don't want them to be added or removed at will, so we designed the ti-community-label-blocker plugin to help us control the permissions for adding or removing such labels.

## Permission design

This plugin is mainly responsible for restricting the add or delete behavior of some labels of Issue or PR. For a label that needs to be controlled, only the operator is allowed to operate normally if he is a trusted user in the configuration or a member of the trusted team.

## Design

The plugin will match the rules of the added or deleted labels according to the rules, and the matched label will be automatically removed or re-added by the plugin if they are added or deleted by untrusted users.

For trusted users, i.e. trusted Github users or members of a trusted Github team, their actions on labels are not affected.

## Parameter Configuration 

| Parameter Name | Type         | Description                      |
| -------------- | ------------ | -------------------------------- |
| repos          | []string     | Repositories                     |
| block_labels   | []BlockLabel | Label with restricted operations |

### BlockLabel

| Parameter Name | Type     | Description                                                        |
| -------------- | -------- | ------------------------------------------------------------------ |
| regex          | string   | Regular expressions for matching label                             |
| actions        | []string | Matching action type, can fill in `labeled` or `unlabeled`, at least one |
| trusted_teams  | []string | Trusted GitHub teams                                               |
| trusted_users  | []string | Trusted GitHub users                                               |
| message        | string   | Feedback hints to the user, empty means no hints                   |

For example:

```yml
ti-community-label-blocker:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
    block_labels:
      - regex: "^status/can-merge$"
        actions: 
          - labeled
          - unlabeled
        trusted_teams: 
          - admins
        trusted_users:
          - ti-chi-bot
        message: "You cannot manually add or delete the status/can-merge label, only the admins team and ti-chi-bot have permission to do so."
```

## Reference Documents

- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/labelblocker)
