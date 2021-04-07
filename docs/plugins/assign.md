# assign

## 设计背景

通过命令让机器人来处理用户指派的事务，同时可以添加一些说明来明确指派原因。

## 设计思路

该插件提供了两个命令，只需要在Issue添加评论即可触发：
- (un)assign: 指派Assignee为指定用户
- (un)cc: 请求指定用户Review

可以通过不指定用户来快速指定自己为对应角色；通过`un`前缀也可以取消该操作。

## 参数配置

无参数

## 参考文档

- [assign doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/assign)

## Q&A

### 为什么支持以非`@`开头的用户名？

> https://github.com/ti-community-infra/tichi/issues/426

当以`@`开头时，github会自动发送邮件给对应用户（评论中@操作）；在bot指派对应角色后会发送一个指派邮件。
为了减少不必要的邮件数量，assign允许非`@`开头指派用户。
