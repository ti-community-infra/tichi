# ti-community-format-checker

## 设计背景

为了能够对 PR 或 commit 的变更内容有更加清晰的描述，在 TiDB 社区当中，我们对 PR 的标题的格式进行了规范和校验，在过去的实践当中，我们使用 Jenkins CI 或 GitHub Actions 的方式进行校验，这样的方式优点在于可以针对特定的仓库进行更好的定制，只需要在分支保护当中添加对应的 [Required Status Check](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/about-protected-branches#require-status-checks-before-merging) 就可以将格式要求作为 PR 的合并条件之一，缺点是每次校验都需要经过一个较长的启动过程，而校验程序的实际运行时间却很短，这其实会对机器资源产生一些浪费。当多个仓库都实行一套统一的格式规范时，通过基于 WebHook 机制实现的 GitHub 机器人来对格式进行校验也许是更好的方式。

## 设计思路

该插件会通过正则表达式的方式对 PR 或 issue 的标题、内容，以及提交中附加的 commit message 的格式进行校验，例如：检查 PR 的标题是否符合 `pkg: what's changed` 这样的格式，如果正则表达式无法匹配，机器人可以通过添加 `do-not-merge/` 类型的标签阻止 PR 合并，也可以通过评论的方式提示 contributor 进行修改。

## 参数配置 

| 参数名                  | 类型                | 说明     |
|----------------------|-------------------|--------|
| repos                | []string          | 配置生效仓库 |
| required_match_rules | RequiredMatchRule | 匹配规则   |

### RequiredMatchRule

| 参数名             | 类型         | 说明                                         |
|-----------------|------------|--------------------------------------------|
| pull_request    | bool       | 是否对 PR 进行校验                                |
| issue           | bool       | 是否对 issue 进行校验                             |
| title           | bool       | 是否对 PR 或 issue 的标题部分进行校验                   |
| branches        | []string   | 指定需要进行校验的分支                                |
| start_time      | *time.Time | 指定规则开始生效的时间，在生效时间之前创建的 PR 或 issue 不会被校验    |
| body            | bool       | 是否对  PR 或 issue 的内容部分进行校验                  |
| commit_message  | bool       | 是否对 PR 当中 commit 的 commit message 部分进行校验   |
| regexp          | string     | 校验时使用的正则表达式                                |
| matched         | string     | 在正则表达式匹配成功，而不是匹配失败时进行对应的操作          |
| missing_message | string     | 当匹配失败时，对 PR 或 issue 进行评论回复，多个规则的提示信息会聚合在一起 |
| missing_label   | string     | 当匹配失败时，对 PR 或 issue 添加的标签                  |
| skip_label      | string     | 指定能够跳过当前检查规则的标签                            |
| trusted_users   | []string   | 允许指定用户可以跳过当前规则检查                           |

匹配规则当中的正则表达式可以通过 [命名分组](https://pkg.go.dev/regexp#Regexp.SubexpNames) 的方式对特定部分进行额外的检查，目前支持的命名分组如下：

- `issue_number`: 机器人会对该部分进行额外的校验，如果发现填写的内容是一个 PR number，而不是一个 issue number，机器人判定为匹配失败

例如：

```yml
ti-community-format-checker:
  - repos:
      - ti-community-infra/test-dev
    required_match_rules:
      - pull_request: true
        title: true
        regexp: "^(\\[TI-(?P<issue_number>[1-9]\\d*)\\])+.+: .{10,160}$"
        branches:
          - main
        start_time: "2021-11-01T12:00:00Z"
        missing_message: |
          Please follow PR Title Format: `[TI-<issue_number>] pkg, pkg2, pkg3: what is changed`
          Or if the count of mainly changed packages are more than 3, use `[TI-<issue_number>] *: what is changed`
        missing_label: do-not-merge/invalid-title
        skip_label: skip-issue
        trusted_users:
          - dependabot[bot]
```

注意: 如果想把匹配规则作为合并前必须通过的检查项，需要将 `missing_label` 设置为 [tide](components/tide) 组件的 `missingLabels` 选项，以及 [tars](plugins/tars) 插件的 `exclude_labels` 选项。

## 参考文档

- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/formatchecker)
