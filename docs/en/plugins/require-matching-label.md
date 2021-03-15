# require-matching-label

## Design Background

When we create an Issue or submit a PR, there are some labels that must be added. For example, for Issue we have to add type/xxx labels to distinguish what type the Issue belongs to. But sometimes we forget to add these key labels, and it takes time and effort to find and add them later. So we want to automate this process and urge people to add relevant labels as soon as possible.

[require-matching-label](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/require-matching-label) detects the labels that must be added by means of a regular labels, **add the corresponding missing labels when they are missing and reply to the comment**.

## Design

This plugin was designed and developed by the Kubernetes community. In addition to implementing the core feature of matching labels based on regular expressions, it also takes into account the fact that there may be automated tools to assist in labeling when an issue or PR is first created, so it supports delayed detection (default is 5 seconds), waiting for a period of time before matching and check whether the labels meet the requirements.

## Parameter Configuration 

| Parameter Name  | Type   | Description                                           |
| --------------- | ------ | ----------------------------------------------------- |
| org             | string | Organization                                          |
| repo            | string | Repository                                            |
| branch          | string | Branch                                                |
| prs             | bool   | Whether to apply to PR                                |
| issues          | bool   | Whether to apply to Issue                             |
| regexp          | string | label regular expressions                             |
| missing_label   | string | When no matching label is found, the label is added   |
| missing_comment | string | When no matching label is found, reply to the comment |
| grace_period    | string | Delayed detection time (default is 5 seconds)         |

For example:

```yaml
- missing_label: needs-sig
  org: pingcap
  repo: tidb
  issues: true
  regexp: ^(sig|wg)/
  missing_comment: |
    There are no sig labels on this issue. Please add an appropriate label by using one of the following commands:
    - `/sig <group-name>`
    - `/wg <group-name>`
```

## Reference Documents

- [require-matching-label doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/require-matching-label)

## Q&A

### Is it possible to check the Issue or PR after it has been created for some time?

Yes, just configure the grace_period parameter. The **default is 5 seconds**.