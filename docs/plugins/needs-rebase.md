# needs-rebase

## 设计背景

通常 GitHub 不会在 PR 冲突时提醒 PR 的作者去解决冲突，这样可能导致我们的 PR 无法被机器人自动合并，需要人为的去提醒解决冲突。另外，我们在 PR 列表中也无法查看哪些 PR 有冲突。

[needs-rebase](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins/needs-rebase) 可以定期的去检测 PR 的冲突情况添加 `needs-rebase` 标签和提醒 PR 作者解决冲突。

## 设计思路

该插件由 Kubernetes 社区设计开发，他们在实现过程中不仅考虑到了需要定期的扫描所有的 PR 去添加或者移除 `needs-rebase` 标签，而且应该针对那些正在活跃并且有回复的 PR 尽快的添加 `needs-rebase` 标签提醒解决冲突。

## 参数配置

无配置

## 参考文档

- [needs-rebase doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins/needs-rebase)

## Q&A

### 机器人在添加 `needs-rebase` 的同时会进行回复，这是不是很打扰 PR 作者？

> https://github.com/ti-community-infra/tichi/issues/408

这是因为 GitHub 在 PR 冲突时不会提醒你冲突了，而且机器人添加 `needs-rebase` GitHub 也不会通知你，所以我们必须显式的通过回复来通知你。
另外，**当你解决冲突之后机器人会自动移除 `needs-rebase` 标签并删除过期无用的回复。**
