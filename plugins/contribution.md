# ti-community-contribution

## 设计背景

在 TiDB 社区中，会有大量的外部贡献者参与到社区当中贡献。我们需要区分内部贡献者和外部贡献者的 PR，优先帮助审阅外部贡献者的 PR，尤其是首次提交 PR 的贡献者，这样才能让外部贡献者有良好的贡献体验。

ti-community-contribution 会帮助我们区分内部贡献者和外部贡献者的 PR，为外部贡献者的 PR 添加 `contribution` 或者 `first-time-contributor` 的标签。

## 设计思路

该插件会根据 PR 作者是否为仓库所在 Org 的成员来添加 `contribution` 标签，另外如果该作者是第一次向该仓库提交 PR 或第一次在 GitHub 上提交  PR，还会为该 PR 添加 `first-time-contributor` 标签。

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

## Q&A

### 为什么是根据是否为 Org 成员来区分是否为外部贡献？

因为如果是 Org 成员说明他至少是 reviewer 以上的身份（**只有 reviewer 以上的身份才会被邀请到 Org**），已经很熟悉PR 流程不太需要帮助。