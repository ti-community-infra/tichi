# ti-community-issue-triage

## 设计背景

在 TiDB 社区现有的版本管理模型当中，TiDB 会同时维护着多个发行版本，严重程度为 critical 或 major 的 bug issue 需要在受此影响且尚在维护的发行版本分支上被修复。

对于 bug issue，我们会通过 `affects/*` 标签来标注其影响发行分支，例如：`affects/5.1` 标签表示该 Bug 会影响 `release-5.1` 分支下的发行版本，但是这样做会存在的一些问题，如果一个 issue 没有标注某个版本的 `affects/*` 标签，我们很难确定该 issue 是否已经经过诊断，还是没有对该 release 分支产生影响。为了防止 bug issue 的修复被遗漏，我们拟定了 [新的 Triage 工作流程](https://github.com/pingcap/community/blob/master/votes/0625-new-triage-method-for-cherrypick.md) 。

在新的流程当中，在合并修复 critical 或 major 严重程度的 bug issue 的 pull request 之前，需要先完成 bug issue 的 triage 过程，以确定 bug issue 会影响的所有 release 分支，当 bug issue triage 完成后，机器人将会自动将 `affects/x.y` 标签转换为对应的 `needs-cherry-pick-release-x.y` 标签添加到 pull request 上，当该 PR 满足其它条件完成合并后，再由机器人自动地创建将修复 PR cherry pick 到各个受影响的 release 分支。

ti-community-issue-triage 插件将被设计来对以上流程进行管控和自动化处理。

## 设计思路

### Issue 方面

对于 critical 或 major 严重程度的 bug issue 而言：

- 当一个新建的 bug issue 被添加上 `severity/critical` 或 `severity/major` 标签后，机器人会根据当前正在维护的 release 分支列表在 issue 上添加对应的 `may-affects/x.y` 标签

- 当 issue 通过诊断，确认该 issue 会影响某个发行分支 `release-x.y` 分支时，可以通过 `/affects x.y` 命令打上对应的 `affects/x.y` 标签，机器人将会同时去掉对应的 `may-affects/x.y` 标签

- 当 issue 上的 `may-affects/x.y` 标签发生变化时，将会重新触发所有与其关联的在 **默认分支** 上（例如：`master`）打开的 pull request 的检查

### Pull request 方面

对于修复相关 bug issue 并与之关联的 pull request 而言，我们会添加一个名为 `check-issue-triage-complete` 的检查项，该检查将会确保 bug issue 在 pull request 合并之前完成 triage。

默认情况下，插件会在合适的时机来触发该检查，如果没有触发成功，Contributor 可以通过 `/run-check-issue-triage-complete` 命令进行手动触发。

插件会根据如下规则来判断 pull request 是否完成 triaged：

- PR 关联的 issue 通过 `Issue Number: ` 行来确定

- PR 关联的 issue 必须包含 `type/*` 标签

- PR 关联的 bug issue 必须包含 `severity/*` 标签

- PR 所关联的标有 `severity/critical` 或 `severity/major` 标签的 bug issue 不能含有任何 `may-affects/x.y` 标签，如果有则认为尚未完成 triage，不满足合并的条件

- PR 如果关联了多个 bug issue，那么这些 issue 都需要满足以上条件

如果 `check-issue-triage-complete` 检查不通过，机器人会自动打上 `do-not-merge/needs-triage-completed` 标签，阻止 PR 的合并。

当 `check-issue-triage-complete` 检查通过时，机器人会去掉 `do-not-merge/needs-triage-completed` 标签，并根据所有关联的 bug issue 的 `affects/x.y` 标签为 PR 自动打上 `needs-cherry-pick-release-x.y` 标签。

## 参数配置 

| 参数名                             | 类型       | 说明                                   |
|---------------------------------|----------|--------------------------------------|
| repos                           | []string | 配置生效仓库                               |
| maintain_versions               | []string | 仓库正在维护的发行分支版本号                       |
| affects_label_prefix            | string   | 标识 issue 影响的发行分支的 label 前缀           |
| may_affects_label_prefix        | string   | 标识 issue 可能影响的发行分支的 label 前缀         |
| linked_issue_needs_triage_label | string   | 标识 PR 所关联 issue 需要 triage 完成的 label  |
| need_cherry_pick_label_prefix   | string   | 标识 PR 需要 cherry-pick 到 release 分支的前缀 |
| status_target_url               | string   | Status check 的详情 URL                 |

例如：

```yml
ti-community-issue-triage:
  - repos:
      - ti-community-infra/test-dev
    maintain_versions:
      - "5.1"
      - "5.2"
      - "5.3"
    affects_label_prefix: "affects/"
    may_affects_label_prefix: "may-affects/"
    linked_issue_needs_triage_label: "do-not-merge/needs-triage-completed"
    need_cherry_pick_label_prefix: "needs-cherry-pick-release-"
    status_target_url: "https://book.prow.tidb.io/#/plugins/issue-triage"
```

## 参考文档

- [0625-new-triage-method-for-cherrypick.md](https://github.com/pingcap/community/blob/master/votes/0625-new-triage-method-for-cherrypick.md)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/issuetriage)

