# assign

## 设计背景

在大型仓库上协作需要将 PR 或 Issue 分配给特定的协作者来跟进，但是如果没有写权限，是无法直接通过 GitHub 页面去分配的。

[assign](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/assign) 可以提供命令让机器人帮助我们分配协作者和请求 reviewer。

## 设计思路

该插件由 Kubernetes 社区设计开发，提供了两个命令：

- `/[un]assign @someone hi-rustin`: 将 Issue/PR 分配或取消分配给 someone 和 hi-rustin。
- `/[un]cc @someone hi-rustin`: 请求或取消 someone 和 hi-rustin review PR。

注意：如果在命令后不指定 GitHub 账号，则默认是自己。

## 参数配置

无参数

## 参考文档

- [assign doc](https://prow.tidb.net/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/assign)

## Q&A

### 为什么支持以非`@`开头的用户名？

> https://github.com/ti-community-infra/tichi/issues/426

当以`@`开头时，GitHub 会自动发送邮件给对应用户，同时在机器人分配或者请求 review 后也会发送一个通知邮件。
为了减少不必要的邮件数量， assign 允许非`@`开头的用户名。

### 为什么机器人提示我无法将一个 Issue 或者 PR `/assign`  给某个人？就算该用户是组织成员也无法被 `/assign`？

GitHub 对分配有一些限制：

- 每个 Issue 或者 PR 最多只能分配给 10 个用户。
- 以下四种用户可以被分配：
    - Issue 或者 PR 作者
    - 在 Issue 或者 PR 上有评论的用户
    - 对该仓库具有写权限的用户
    - 对该仓库有读权限的组织成员（**注意：公开仓库的可见性与协作者的读权限不同，此处是指在仓库权限设置中显式的被添加为有读权限的协作者**）
    
另外也可以参考 GitHub 对分配的[说明文档](https://docs.github.com/cn/issues/tracking-your-work-with-issues/managing-issues/assigning-issues-and-pull-requests-to-other-github-users)。