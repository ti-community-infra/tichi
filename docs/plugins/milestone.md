# milestone

## 设计背景

通过命令让机器人来处理指派milestone的事务，同时可以添加一些说明来明确指派原因。

## 设计思路

通过在评论中`/milestone xxx`的命令来给Issue配置milestone。使用`/milestone clear`来清除配置。

***只有milestone维护人员才可以使用该命令。***

## 参数配置

配置中`repo_milestone`下，字典的key为对应的repo、value为对应的维护人员信息。当key为空时为默认维护人员。

|参数名                    |类型       |说明 |
|-------------------------|----------|----|
|maintainers_id           |维护人员ID  |    |
|maintainers_team         |维护团队    |    |
|maintainers_friendly_name|维护团队昵称 |    |
 
可以使用以下接口获取您的milestone维护团队的GithubID，您可能需要手动指定`page`参数

```shell
curl -H "Authorization: token <token>" "https://api.github.com/orgs/<org-name>/teams?page=N"
```

相关配置示例：

```yaml
repo_milestone:
  ti-community-infra/test-dev:
    maintainers_id: 4300209
    maintainers_team: bots-maintainers
    maintainers_friendly_name: Robots Maintainers
```

## 参考文档

- [milestone doc](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/milestone)

## Q&A

> 暂无
