# needs-rebase

## 设计背景

Github不会在提交commit时提醒代码冲突，该插件主要用于通过`needs-rebase`标签提醒用户PR是否有代码冲突。

## 设计思路

该插件会监听所有的PR commit改动，并定时扫描全部open状态的PR来确定PR是否有代码冲突。

## 参数配置

需要在`external_plugins`配置中增加对应repo的needs-rebase插件，示例如下：

```yaml
external_plugins:
  ...
  ti-community-infra/test-dev:
    - name: needs-rebase
      events:
        - issue_comment
        - pull_request
  ...
```

## 参考文档

- [needs-rebase doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins/needs-rebase)

## Q&A

### 'PR needs rebase'消息过于分散注意力，是否有必要？

> https://github.com/ti-community-infra/tichi/issues/408

当冲突解决后bot会自动删除消息。
