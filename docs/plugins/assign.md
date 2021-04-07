# assign

## 设计背景

在大型仓库上协作需要将 PR 或 Issue 分配给特定的协作者来跟进，但是如果没有写权限，是无法通过页面去分配的。

assign 可以提供命令让机器人帮助我们分配协作者和请求 reviewer。

## 设计思路

该插件提供了两个命令，只需要在 Issue 或 PR 添加评论即可触发：
- (un)assign: 指派 Assignee 为指定用户
- (un)cc: 请求指定用户 Review

可以通过不指定用户来快速指定自己为对应角色；通过`un`前缀也可以取消该操作。

## 参数配置

无参数

## 参考文档

- [assign doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/assign)

## Q&A

### 为什么支持以非`@`开头的用户名？

> https://github.com/ti-community-infra/tichi/issues/426

当以`@`开头时，github 会自动发送邮件给对应用户（评论中@操作）；在 bot 指派对应角色后会发送一个指派邮件。
为了减少不必要的邮件数量， assign 允许非`@`开头指派用户。
