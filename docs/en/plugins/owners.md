# ti-community-owners

## Design Background

The design owners are primarily for the ti-community-lgtm and ti-community-merge plugins, and in the Kubernetes community they use the [OWNERS](https://github.com/kubernetes/test-infra/blob/master/OWNERS) file to define permissions, where reviewers and approvers for the current directory and subdirectories are specified. But the TiDB community has now relied on the community's previous [Bot](https://github.com/pingcap-incubator/cherry-bot) for a long time. The community has worked out a collaborative mechanism for using the previous Bot. Adopting the Kubernetes community's mechanism directly would be very costly to learn and unconvincing.

So we decided to develop a permission control service that fits the current collaborative model of the TiDB community, based on the current [SIG](https://github.com/pingcap/community) architecture of the TiDB community to define permissions for each PR.

## Permission design

Based on the current design of TiDB's SIG, the permissions are divided as follows:

- committers(**Can merge a pull request**)
  - maintainers
  - techLeaders
  - coLeaders
  - committers
- reviewers(**Can approve a pull request**)
  - maintainers
  - techLeaders
  - coLeaders
  - committers
  - reviewers

In the TiDB community collaboration process, **a PR is usually reviewed several times before it is merged**. Therefore, it is important to define the number of `LGTM` required for each PR in this service. You can also specify the label prefix for the number of `LGTM` required by the PR in the configuration. owners will automatically read the required `LGTM` number from the label.

## Design

In order to implement the division of permissions described above, we decided to adopt the RESTFUL interface to define the permissions for each PR.

API: `/repos/:org/:repo/pulls/:number/owners`

The owners will look for labels starting with `sig/` in the current PR and then look for information about the SIG. Finally, owners are generated based on the information of the SIG obtained.

However, there are some special cases where the corresponding SIG cannot be found:
- Some modules do not have clear SIG affiliations at the moment: reviewers and committers using all SIGs of the TiDB community
- Some small repositories belong directly to a SIG: Support for configuring a default SIG for this repository
- Some repositories whose PR is independent of SIG: support for using [GitHub permissions](https://docs.github.com/en/organizations/managing-access-to-your-organizations-repositories/repository-permission-levels-for-an-organization)
  - committers(**Can merge a pull request**)
    - Admin Permission
    - Maintain Permission
    - Write Permission
  - reviewers(**Can approve a pull request**)
    - Triage Permission

Note: Because maintainers are not attached to any SIG, they will be fetched directly from the GitHub team via a configuration item.

## Parameter Configuration 

| Parameter Name            | Type                    | Description                                                                                                                                                          |
|---------------------------|-------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| repos                     | []string                | Repositories                                                                                                                                                         |
| sig_endpoint              | string                  | Address of the RESTFUL API for obtaining SIG information                                                                                                             |
| default_sig_name          | string                  | Set the default SIG for this repository                                                                                                                              |
| default_require_lgtm      | int                     | Set the default number of lgtm required for this repository                                                                                                          |
| require_lgtm_label_prefix | string                  | The plugin supports specifying the number of lgtm required for the current PR by label, and this option is used to set the prefix of the relevant label              |
| use_github_team           | bool                    | Use the specified GitHub Team as reviewer teams and committer teams                                                                                                  |
| committer_teams           | []string                | Specify a list of GitHub team names whose members can serve as committers                                                                                            |
| reviewer_teams            | []string                | Specify a list of GitHub team names whose members can serve as reviewers                                                                                             |
| use_github_permission     | bool                    | Use GitHub permissions                                                                                                                                               |
| branches                  | map[string]BranchConfig | Branch granularity parameters configuration, map structure key is the branch name, the configuration of the branch will override the configuration of the repository |

### BranchConfig

| Parameter Name        | Type     | Description                                                                             |
|-----------------------|----------|-----------------------------------------------------------------------------------------|
| default_require_lgtm  | int      | Set the default number of lgtm required for the branch                                  |
| use_github_team       | bool     | Use the specified GitHub Team as reviewer teams and committer teams                     |
| committer_teams       | []string | Set up a list of GitHub team names for the branch whose members can serve as committers |
| reviewer_teams        | []string | Set up a list of GitHub team names for the branch whose members can serve as reviewers  |
| use_github_permission | bool     | Use GitHub permissions                                                                  |

For example:

```yml
ti-community-owners:
  - repos:
      - ti-community-infra/test-live
      - ti-community-infra/tichi
      - ti-community-infra/ti-community-bot
      - ti-community-infra/ti-challenge-bot
    sig_endpoint: https://bots.tidb.io/ti-community-bot
    require_lgtm_label_prefix: require/LGT
    use_github_team: true
    reviewer_teams:
      - bots-reviewers
    committer_teams:
      - bots-committers
    branches:
      release:
        default_require_lgtm: 2
        use_github_permission: true
        reviewer_teams:
          - bots-reviewers
        committer_teams:
          - qa-release-team
```

## Q&A

### How can I check the current PR permissions?

 Directly check the GitHub-compliant RESTFUL API, for example: [ti-community-infra/test-dev/pulls/179](https://prow.tidb.io/ti-community-owners/repos/ti-community-infra/test-dev/pulls/179/owners)
