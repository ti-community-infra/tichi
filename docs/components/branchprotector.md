# branchprotector

## 设计背景

branchprotector 是根据特定策略自动更新 [GitHub 仓库分支保护](https://help.github.com/articles/about-protected-branches/) 配置的组件。

## 设计思路

通过 yaml 配置文件来设置 GitHub 仓库的分支保护策略，由 branchprotector 组件通过 [update-branch-protection API](`https://docs.github.com/en/rest/reference/repos#update-branch-protection`) 定期（[每半小时](https://github.com/ti-community-infra/configs/blob/main/prow/cluster/branchprotector.yaml#:~:text=spec%3A-,schedule)）地将配置文件中策略同步到对应的代码仓库的分支保护当中。 

通过 branchprotector 组件来对代码仓库的分支保护配置进行统一管理，能够将分支保护配置更好地向社区公开，协作者可以通过提交 Pull Request 的方式来对配置文件进行修改，配置的修改需要经过项目维护团队与社区基础设施团队的审阅，从而避免分支保护配置的修改导致错误（例如：与协作机器人的配置相冲突）。另外，我们还通过 Git History 查看到配置文件的变更历史。

## 配置说明

branchprotector 的配置位于 Prow 的主要配置文件 [config.yaml](https://github.com/ti-community-infra/configs/blob/main/prow/config/config.yaml#:~:text=branch-protection) 当中。

### 配置项

你可以在 GitHub 的 [Branch Projection API](https://developer.github.com/v3/repos/branches/#update-branch-protection) 文档当中查看到完成的配置项列表

```yaml
branch-protection:
  # 在此处可以添加默认策略
  orgs:
    foo:
      # 在此处可以添加 foo Org 的策略
      protect: true  # 启用分支保护
      enforce_admins: true  # 将规则对仓库管理员应用
      required_linear_history: true  # 强制要求 Git Commit 保持线性的提交历史
      allow_force_pushes: true  # 允许 force pushes 方式推动代码到受保护的分支
      allow_deletions: true  # 允许删除受保护的分支
      required_pull_request_reviews:
        dismiss_stale_reviews: false # 忽略过时的 Review，不作为有效的 Review
        dismissal_restrictions: # 忽略指定用户或指定 Team 的 Review，不作为有效的 Review
          users:
          - her
          - him
          teams:
          - them
          - those
        require_code_owner_reviews: true  # 要求 PR 经过相关的 Code Owners 进行 Review
        required_approving_review_count: 1 # 要求来自具有 Write 以上权限的协作者的 Approval 数量
      required_status_checks:
        strict: false # 是否要求 PR 的 Base 保持最新
        contexts: # 合并前必须通过 Status Check，对仓库启用的 PreSubmit Job 会默认作为 Required Context
        - foo
        - bar
      restrictions: # 允许指定的用户或指定的团队将 PR Push 到受保护的分支
        users:
        - her
        - him
        teams:
        - them
        - those
```

### 作用范围

分支保护策略支持全局、Org、Repo、Branch 四种级别的配置，子配置会覆盖父配置，覆盖的规则为：

- 当一个子配置的值为 null 或被忽略时，会继承父配置的值
- 对于列表类型的值（例如：`contexts`），父配置与子配置会合成一个并集作为最终的值
- 对于布尔类型或整数型的值（例如：`protect`）, 子配置的值会替代父配置当中的值

```yaml
branch-protection:
  # Protect unless overridden
  protect: true
  # If protected, always require the cla status context
  required_status_checks:
    contexts: ["cla"]
  orgs:
    unprotected-org:
      # Disable protection unless overridden (overrides parent setting of true)
      protect: false
      repos:
        protected-repo:
          protect: true
          # Inherit protect-by-default config from parent
          # If protected, always require the tested status context
          required_status_checks:
            contexts: ["tested"]
          branches:
            secure:
              # Protect the secure branch (overrides inhereted parent setting of false)
              protect: true
              # Require the foo status context
              required_status_checks:
                contexts: ["foo"]
    different-org:
      # Inherits protect-by-default: true setting from above
```

## 参考文档

- [README](https://github.com/kubernetes/test-infra/blob/master/prow/cmd/branchprotector/README.md)
- [代码实现](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/branchprotector)

## Q&A

### `branch-protection` 的 `required_status_checks` 配置与 tide 的 `required-contexts` 配置的关系是什么？

`branch-protection` 的配置会同步到代码仓库的分支保护当中，其中设置的 `required_status_checks` 会要求 PR 在合并之前必须通过指定的 Status Check，即便是仓库管理员也无法在 Required Status Check 全部通过前合并 PR。

而 `required-contexts` 配置是作为 tide 自动合并 PR 的条件之一，在 `required-contexts` 没有全部通过之前，tide 不会尝试合并该 Pull Request，但是该配置并不会阻止其他人对 PR 进行合并。

一般情况下，`required_status_checks` 与 `required-contexts` 两项配置应该是一致的，在 tide 的配置当中可以通过 `from-branch-protection: true` 选项直接将 `required_status_checks` 配置作为 tide 的 `required-contexts` 配置。

而在一些情况下，一些 CI 测试并不总能稳定执行，如果将其作为 `required_status_checks`，包括仓库管理员在内的协作者都无法对测试失败的 PR 进行合并。此时，如果只将不稳定测试的 CI 作为 `required-contexts`，当 CI 失败导致 tide 无法自动合并 PR 时，协作者仍然有权限对 PR 进行合并。