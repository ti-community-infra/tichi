# ti-community-label

## Design Background 

In the TiDB community, Issue and PR both have a large number of labels, and we were confused in the process of using them, and many labels were not clearly defined and categorized. The original bot only supported a labeling command like `/label`. This led to a lot of pain in using the command, because many times the labels would have a prefix for the category and the labels would be long, and it was always hard to remember the labels when using them.

ti-community-label takes a different tack, as the plugin supports labeling by category. For example, for labels like `type`, we can use commands like `/[remove-]type bug` to more semantically label or unlabel an Issue or PR as `type/bug`. In addition, we also keep the `/[remove-]label` command to label or unlabel uncategorizable items.

## Permission design

This plugin is mainly responsible for adding labels to Issues or PRs, so we set the permissions to allow all GitHub users to use this feature.

## Design

The plugin mainly refers to the Kubernetes label plugin design and extends on it to support custom label prefixes (categories) for each repository. This allows you to organize labels for repositories in categories during use.

In addition, in the process of implementing the plugin, you should pay attention to:**Because the plugin permissions are set loosely, so you can only add labels that have been created by the repository, otherwise there is a risk of being labeled with something useless.**

## Parameter Configuration

| Parameter Name    | Type     | Description                                                                                                              |
| ----------------- | -------- | ------------------------------------------------------------------------------------------------------------------------ |
| repos             | []string | Repositories                                                                                                             |
| additional_labels | []string | Uncategorized labels                                                                                                     |
| prefixes          | []string | Category Prefix                                                                                                          |
| exclude_labels    | []string | Some labels that you do not want to be added or removed by the plugin (e.g. some labels that only allow bots to operate) |

For example:

```yml
ti-community-label:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/prow-configs
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
    prefixes:
      - type
      - status
    additional_labels:
      - 'help wanted'
      - 'good first issue'
    exclude_labels:
      - 'status/can-merge'
```

## Reference Documents

- [command help](https://prow.tidb.io/command-help?repo=ti-community-infra%2Ftichi#type)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/label)

## Q&A

### Why is there no response to adding tags using this feature?

Please check if the label exists in the repository, the plugin will only add labels that have already been created by the repository.