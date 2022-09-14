# milestone

## 设计背景

在大型仓库上我们会使用 milestone 来追踪 PR 和 Issue 的进度，但是 GitHub 限制只有写权限的协作者才能为 Issue/PR 添加 milestone。

[milestone](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/milestone) 可以提供命令让机器人添加对应的 milestone。

## 权限设计

该插件主要负责添加 milestone，所以只能让 milestone 的管理团队来使用该命令。

## 设计思路

该插件由 Kubernetes 社区设计开发，提供了两个命令：

- `/milestone v1.3.2 v1.4` 添加 milestone v1.3.2 和 v1.4。
- `/milestone clear` 清除 Issue/PR 上所有的 milestones。

注意：只有 milestone 管理团队才可以使用该命令。

## 参数配置

| 参数名                    | 类型   | 说明           |
| ------------------------- | ------ | -------------- |
| maintainers_id            | int    | GitHub 团队 ID |
| maintainers_team          | string | GitHub 团队名  |
| maintainers_friendly_name | string | 团队昵称       |

例如：

```yaml
repo_milestone:
  ti-community-infra/test-dev:
    maintainers_id: 4300209
    maintainers_team: bots-maintainers
    maintainers_friendly_name: Robots Maintainers
```

## 参考文档

- [milestone doc](https://prow.tidb.net/plugins?repo=ti-community-infra%2Ftichi)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/milestone)

## Q&A

### 我如何才能获取我 GitHub 团队的 ID？

```sh
curl -H "Authorization: token <token>" "https://api.github.com/orgs/<org-name>/teams?page=N"
```

通过以上 API 可以获取该组织下所有的 GitHub 团队详细信息。
