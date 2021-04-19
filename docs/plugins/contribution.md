# ti-community-contribution

## 设计背景

我们希望为 TiDB 社区中暂不属于当前仓库所在 org 的贡献者提出的 PR 赋予额外的能见度，以支持筛选并处理这部分贡献。

ti-community-contribution 会帮助我们判断当前 PR 是否由仓库所在 org 的成员提出，如果不是，为该 PR 添加 `contribution` 或者 `first-time-contributor` 的标签。

## 设计思路

该插件会根据 PR 作者是否为仓库所在 org 的成员来添加 `contribution` 标签，另外如果该作者是第一次向该仓库提交 PR 或第一次在 GitHub 上提交  PR，还会为该 PR 添加 `first-time-contributor` 标签。

## 参数配置 

| 参数名  | 类型     | 说明                   |
| ------- | -------- | ---------------------- |
| repos   | []string | 配置生效仓库           |
| message | string   | 添加标签之后回复的消息 |

例如：

```yml
ti-community-merge:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
      - tikv/pd
    message: "Thank you for your contribution, we have some references for you."
```

## 参考文档

- [RFC](https://github.com/ti-community-infra/rfcs/blob/main/active-rfcs/0001-contribution.md)
- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/contribution)
- 
