# ti-community-label-blocker

## 设计背景 

在整个 PR 工作流当中，一些标签会被机器人识别和使用，例如：通过根据是否带有 `status/can-merge` 标签来判断该 PR 是否能够被合并。对于这些较为敏感的标签，我们不希望它们被别人随意的添加或删除，因此设计了 ti-community-label-blocker 这个插件来帮助我们对这类标签的添加或删除操作进行权限控制。

## 权限设计

该插件主要负责的对 issue 或者 PR 的一些标签的添加或删除行为进行限制，对于一个需要被拦截的标签，只有在配置当中属于

## 设计思路

插件允许用户在配置当中针对指定的一个或多个仓库添加拦截标签的规则。

插件根据这些规则对添加或删除的标签进行规则匹配，匹配到的标签如果是被非信任用户添加，会被插件自动移除或重新添加。

对于信任用户，即信任的 Github user 或信任的 Github team 当中的成员，他们对标签的操作不会受到影响。

## 参数配置

| 参数名             | 类型         | 说明                                     |
| ----------------- | ------------ | --------------------------------------- |
| repos             | []string     | 配置生效仓库                              |
| labels            | []BlockLabel | 额外的无法分类的 labels                   |

### BlockLabel

| 参数名               | 类型       | 说明                                              |
| -------------------- | -------- | ------------------------------------------------ |
| regex                | string   | 匹配标签的正则表达式                                 |
| actions              | []string | 匹配的 action 类型，不能为空, 可填 labeled、unlabeled |
| trusted_teams        | []string | 设置信任的 GitHub team                             |
| trusted_users        | []string | 设置信任的 GitHub user                             |

例如：

```yml
ti-community-label-blocker:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
    labels:
      - regex: "(?mi)^status/can-merge$"
        actions: 
          - labeled
          - unlabeled
        trusted_teams: 
          - admins
        trusted_users:
          - ti-chi-bot
```

## 参考文档

- [代码实现](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/labelblocker)
