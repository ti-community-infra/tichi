# dco (Developer Certificate of Origin)

## Background

> DCO is the abbreviation of [Developer Certificate of Origin](https://developercertificate.org/), which was established in 2004 by the Linux Foundation. Compared to signing a CLA agreement, contributors do not need to read lengthy legal provisions, but only need to sign the email address in the commit message in the format of `Signed-off-by: Random <random@example.com>` when submitting the code. , To alleviate the hindrance of developers' contribution.


In order to ensure that code contributors sign all commits when submitting a Pull Request, the managers of open source warehouses usually check all the commits of the PR through CI checks or GitHub Bot.

For example: [probot/dco](https://github.com/probot/dco) robot which is widely used for DCO Check. Unfortunately, the original team of the DCO robot no longer maintains the project ([probot/dco#162](https://github.com/probot/dco/issues/162#issuecomment-941149056) ), and GitHub have shutdown the server of the Probot App. 

If you want to continue using this Probot App, you need to deploy it yourself. As an alternative, we can directly open the dco plugin on the repositories where TiChi has been deployed to implement the same check.

## Design


Contributors can add the `-s` parameter to the command line that submits the Commit, and the git command will automatically add the signing information.

```bash
git commit -s -m'This is my commit message'
```

```bash
This is my commit message

Signed-off-by: Random <random@example.com>
```

When contributors submit code through PR, the robot will check the submitted commit. If it finds that one of the commit messages does not contain the `Signed-off-by:` field, the robot will list out no-signed commits through comment.

Contributors can use [`git commit --amend`](https://docs.github.com/en/github/committing-changes-to-your-project/creating-and-editing-commits/changing-a-commit-message) and other methods to resign-off the commits.

When all commits have been signed off, the robot will change the status of `dco` to passed.

![doc_all_commits_signed_off](https://user-images.githubusercontent.com/5086433/143772523-3eeaf9f0-3021-4eb9-9c9d-81f2ce7878cc.png)

## Configuration

The DCO check for Org members or collaborators can be skipped through configuration.

```yaml
dco:
    org/repo:
        # Skip the DCO check for project collaborators
        skip_dco_check_for_collaborators: true
        # Skip the DCO check for project members
        skip_dco_check_for_members: true
        # Specify the members of Org to skip the DCO check. When the skip_dco_check_for_members option is enabled, the members of the organization where the current warehouse is located are skipped by default
        trusted_org: org
```

Under normal circumstances, open source warehouses using DCO agreements will make all commits in the PR must be signed as one of the necessary conditions for PR merger. To achieve this feature, you can use GitHub's branch protection mechanism to use the status of `dco` as [Required Context Status](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/about-protected-branches#require-status-checks-before-merging).

In the TiDB community, we use the [branchprotector](components/branchprotector.md) component to manage branch protection. You can add `dco` to the corresponding repository's [`required_status_checks`](https://github.com/ti-community-infra/configs/blob/main/prow/config/config.yaml#:~:text=branch-protection) Among the configuration items, the robot will be in the PR merged *half an hour * The warehouse branch protection is automatically set up inside.

```yaml
branch-protection:
  orgs:
    ti-community-infra:
      repos:
        test-dev:
          branches:
            master:
              protect: true
              required_status_checks:
                contexts:
                  -"dco"
                  # other status check...
```

## Reference Documents

- [dco doc](https://prow.tidb.net/plugins?repo=ti-community-infra%2Ftichi)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/dco)