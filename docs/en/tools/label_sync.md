# label-sync

## Design Background

On a large GitHub repository, there can be hundreds of labels, and maintaining them manually can be a huge burden. So the Kubernetes community has designed and developed [label_sync](https://github.com/kubernetes/test-infra/tree/master/label_sync) to automate the creation, modification, and maintenance of labels.

## Design

Implementing this tool takes into account not only that we need to create new labels, but also that old labels may need to be migrated for maintenance. So the Kubernetes community has added some new fields to the label definition to handle these situations:

| Parameter Name   | Type       | Description                        |
| ---------------- | ---------- | ---------------------------------- |
| name             | string     | The name of the label              |
| color            | string     | Color of label                     |
| description      | string     | Description of label               |
| target           | string     | Target of label: prs/issues/both   |
| prowPlugin       | string     | Which prow plugin adds this label  |
| isExternalPlugin | bool       | Whether added by external plugins  |
| addedBy          | string     | Who can add                        |
| previously       | []Label    | Old label names before migration   |
| deleteAfter      | *time.Time | How long before deleting the label |

With some extensions to the labels, we define the basic information of these labels and have the flexibility to migrate and maintain all of them.

For example:

```yaml
  - name: dead-label
    color: cccccc
    description: a dead label
    target: prs
    addedBy: humans
    deleteAfter: 2020-01-01T13:00:00Z
```

## Parameter Configuration 

For example:

```yaml
default: # Apply these default labels to all repos
  labels:
    - name: priority/P0
      color: red
      description: P0 Priority
      previously:
      - color: blue
        name: P0
        description: P0 Priority
    - name: dead-label
      color: cccccc
      description: a dead label
      target: prs
      addedBy: humans
      deleteAfter: 2020-01-01T13:00:00Z
repos: # Individual configuration for each repo
  pingcap/community:
    labels:
      - color: 00ff00
        description: Indicates that a PR has LGTM 1.
        name: status/LGT1
        target: prs
        prowPlugin: ti-community-lgtm
        addedBy: prow
      - color: 00ff00
        description: Indicates that a PR has LGTM 2.
        name: status/LGT2
        target: prs
        prowPlugin: ti-community-lgtm
        addedBy: prow
      - color: 0ffa16
        description: Indicates a PR has been approved by a committer.
        name: status/can-merge
        target: prs
        prowPlugin: ti-community-merge
        addedBy: prow
```

## Reference Documents

- [README](https://github.com/kubernetes/test-infra/blob/master/label_sync/README.md)
- [code](https://github.com/kubernetes/test-infra/tree/master/label_sync)

## Q&A

### When will these labels be updated?

Each update of [labels file](https://github.com/ti-community-infra/configs/blob/main/prow/config/labels.yaml) will trigger a [Prow Job](https://github.com/ti-community-infra/configs/blob/fa8e01168a1734a3e372c8e552aef27c102c8f60/prow/jobs/ti-community-infra/configs/configs-postsubmits.yaml#L61) to be updated.