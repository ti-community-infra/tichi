# label-sync

## 设计背景

在 GitHub 的大型仓库上可能会存在多达上百个 labels，手动维护这些 labels 是个很大的负担。所以 Kubernetes 社区设计开发了 [label_sync](https://github.com/kubernetes/test-infra/tree/master/label_sync) 来自动化 label 的创建、修改和维护。

## 设计思路

实现该工具不仅要考虑到我们需要新建 label，而且要考虑到旧的 label 可能需要迁移维护等情况。所以 Kubernetes 社区在定义 label 时新增了一些字段来处理这些情况：

| 参数名           | 类型       | 说明                              |
| ---------------- | ---------- | --------------------------------- |
| name             | string     | label 的名称                      |
| color            | string     | label 的颜色                      |
| description      | string     | label 的描述                      |
| target           | string     | label 的生效目标：prs/issues/both |
| prowPlugin       | string     | 哪一个 prow plugin 添加这个 label |
| isExternalPlugin | bool       | 是否为外部插件添加                |
| addedBy          | string     | 谁可以添加                        |
| previously       | []Label    | 迁移之前旧的 labels               |
| deleteAfter      | *time.Time | label 多久以后删除                |

通过对 label 的一些扩展，我们定义清楚了这些 label 的基本信息并且能够灵活的迁移维护 label。

例如：

```yaml
  - name: dead-label
    color: cccccc
    description: a dead label
    target: prs
    addedBy: humans
    deleteAfter: 2020-01-01T13:00:00Z
```

## 配置说明

例如：

```yaml
default: # default 将这些标签应用给所有的 repos
  labels:
    - name: priority/P0
      color: red
      description: P0 Priority
      previously:
      - color: blue
        name: P0
        description: P0 Priority
    - name: dead-label
      color: cccccc
      description: a dead label
      target: prs
      addedBy: humans
      deleteAfter: 2020-01-01T13:00:00Z
repos: # 针对每个 repo 进行单独配置
  pingcap/community:
    labels:
      - color: 00ff00
        description: Indicates that a PR has LGTM 1.
        name: status/LGT1
        target: prs
        prowPlugin: ti-community-lgtm
        addedBy: prow
      - color: 00ff00
        description: Indicates that a PR has LGTM 2.
        name: status/LGT2
        target: prs
        prowPlugin: ti-community-lgtm
        addedBy: prow
      - color: 0ffa16
        description: Indicates a PR has been approved by an committer.
        name: status/can-merge
        target: prs
        prowPlugin: ti-community-merge
        addedBy: prow
```

## 参考文档

- [README](https://github.com/kubernetes/test-infra/blob/master/label_sync/README.md)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/label_sync)

## Q&A

### 这些 label 什么时候更新？

目前为一个[定时任务](https://github.com/tidb-community-bots/configs/blob/main/prow/cluster/label_sync.yaml)，每小时的第 17 分钟尝试更新。