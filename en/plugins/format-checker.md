# ti-community-format-checker

## design background

In order to be able to have a clearer description of the changes to the PR or commit, in the TiDB community, we have standardized and verified the format of the PR title. In the past practice, we used Jenkins CI or GitHub Actions to check this, the advantage of this method is that it can be better customized for a specific repository. You only need to add the corresponding [required status check](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/about-protected-branches#require-status-checks-before-merging) then the format requirements can be used as PR merge conditions, one of the disadvantages is that each time verification requires a long startup process, but the effective running time of the verification program is very short, which actually causes some waste of machine resources. When multiple repositories implement a unified format specification, it may be a better way to verify the format through a GitHub robot based on the WebHook mechanism.

## Design

The plugin will use regular expressions to verify the title and content of the PR or issue, and the format of the commit message attached to the commit. For example, check whether the title of the PR conforms to the format of `pkg: what's changed`, if regexp cannot be matched. Robots can prevent PR from merging by adding the `do-not-merge/*` label, or they can prompt the contributor to make changes through comments.

## Parameter Configuration

| Parameter name       | Type              | Description    |
|----------------------|-------------------|-- -------------|
| repos                | []string          | Repositories   |
| required_match_rules | RequiredMatchRule | matching rules |

### RequiredMatchRule

| Parameter name  | Type   | Description                                                                                                                          |
|-----------------|--------|-- - ---------------------------------------------------------------------------------------------------------------------------------|
| pull_request    | bool   | Whether to verify the PR                                                                                                             |
| issue           | bool   | Whether to verify the issue                                                                                                          |
| title           | bool   | Whether to verify the title part of the PR or issue                                                                                  |
| body            | bool   | Whether to verify the content of PR or issue                                                                                         |
| commit_message  | bool   | Whether to verify the commit message part of commit in PR                                                                            |
| regexp          | string | Regular expression used in verification                                                                                              |
| missing_message | string | When the match fails, comment and reply to the PR or issue, and the prompt information of multiple rules will be aggregated together |
| missing_label   | string | The label added to PR or issue when the match fails                                                                                  |

The regular expressions in the matching rules can be used to perform additional checks on specific parts by [named group](https://pkg.go.dev/regexp#Regexp.SubexpNames). The currently supported named groups are as follows:

- `issue_number`: The robot will perform additional checks on this part. If it finds that the content filled in is a PR number instead of an issue number, the robot judges that the match failed

For example:

```yml
ti-community-format-checker:
  - repos:
      - ti-community-infra/test-dev
    required_match_rules:
      - pull_request: true
        title: true
        regexp: "^(\\[TI-(?P<issue_number>[1-9]\\d*)\\])+.+: .{10,160}$"
        missing_message: |
          Please follow PR Title Format: `[TI-<issue_number>] pkg, pkg2, pkg3: what is changed`
          Or if the count of mainly changed packages are more than 3, use `[TI-<issue_number>] *: what is changed`
        missing_label: do-not-merge/invalid-title
```

## Reference Documents

- [code](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins/fotmatchecker)
