# [Tide](https://github.com/kubernetes/test-infra/tree/master/prow/tide)

Tide is a core component of Prow that focuses on managing the GitHub PR pool with a few given conditions. It will automatically re-detect PRs that meet the conditions and automatically merge them when they pass the test.

It has the following features:
- Automatically run batch tests and merge multiple PRs together when possible. (This feature is disabled if you do not use Prow's CI)
- Make sure to test the PR against the most recent base branch commit before allowing the PR to be merged. (This feature is disabled if you do not use Prow's CI)
- Maintain a GitHub status context that indicates whether each PR is in the PR pool or what requirements are missing. (This is a status similar to what other CIs report in their PRs, and the current status of the PR is specified in the message for that status)
- Support for blocking PR merges to a single branch or an entire repository using a special GitHub Label.
- Prometheus metrics.
- Supports having "optional" state contexts that are not mandatory for merging.
- Provides real-time data about the current PR pool and attempted merge history, which can be displayed in [Deck](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/deck), [Tide Dashboard](https://prow.tidb.io/tide), [PR Status](https://prow.tidb.io/pr), and [Tide History](https://prow.tidb.io/tide-history).
- Effectively scales so that a single instance with a single bot token can provide merge automation for dozens of organizations and repositories that meet the merge criteria. Each different `org/repo:branch` combination defines a merge pool that does not interfere with each other, so merging only affects other PRs in the same branch.
- Provides configurable merge modes ('merge', 'squash', or 'rebase').

## Tide Merger Rules

An example of the rules for Tide PR merging:

```yaml
tide:
  merge_method:
    pingcap/community: squash # The merge method for this repository is squash.

  queries:
  - repos:
    - pingcap/community
    labels:
    - status/can-merge # PRs for this repository can only be merged if they are labeled with status/can-merge.
    missingLabels:
    # If the PR of the repository has the following labels, the PR will not be merged.
    - do-not-merge
    - do-not-merge/hold
    - do-not-merge/work-in-progress
    - needs-rebase

  context_options:
    orgs:
      pingcap:
        repos:
          community:
            # Require that all PRs for all branches of this repository pass the CI for license/cla.
            required-contexts:
             - "license/cla"
            branches:
              master:
                # Require that all PRs for the master of this repository must pass the Sig Info File Format CI.
                required-contexts:
                - "Sig Info File Format"
```

After we submit a PR, Tide will periodically check to see if each PR meets the above criteria. For the pingcap/community repository, after you submit a PR, Tide will check your PR every minute to see if the PR's label and CI have met the merge requirements.

## Tide in the TiDB Community

Tide works mostly fine in the TiDB community, but there's still a tricky issue (**other communities haven't solved it yet either**):

- PR1: Rename bifurcate() to bifurcateCrab()
- PR2: Use bifurcate()
  
In this case, both PRs will be tested with the current master as the base branch, and both PRs will pass. However, once PR1 is merged into the master branch first, and the second PR is merged (because the test also passes), it causes a master error that `bifurcate` is not found.

I will describe how to solve this problem in the recommended PR workflow.

**The Kubernetes community does not currently have this issue because if you use Prow's CI system Tide will automatically have the latest master as a base for testing**.


### Q&A

#### Where can I find my PR status?

Get PR status from these places:

- The CI status context below the PR. The status either tells you that your PR is in the merge pool now, or tells you why it is not. Clicking on the details will jump you to the [Tide Dashboard](https://prow.tidb.io/tide). For example: ![example](https://user-images.githubusercontent.com/29879298/98230629-54037400-1f96-11eb-8a9c-1144905fbbd5.png ':size=70%')
- In [PR status](https://prow.tidb.io/pr), you will have a card for each of your PRs, each showing the test results and merge requirements. (recommended use)
- In the [Tide Dashboard](https://prow.tidb.io/tide), the status of each merge pool is displayed, allowing you to see what Tide is currently doing and where PR is in the redetection queue.

#### Is my PR in the merge queue?

If the status of Tide is successful (green) then it is already in the merge pool, if it is unsuccessful (yellow) then it will not be in the merge pool.

#### Why is my PR not in the merge queue?

If you have just updated your PR, please wait a bit to give Tide time to detect it (one minute by default).

Whether your PR is in the queue or not is determined by the following two requirements:
- Check that your PR label meets the requirements.
- Check that the required CIs have all passed.

#### Why is the status of Tide still Penning even though PR is merged?

This is because in some cases it may detect that the PR has been met and merge it, but then merge it before it has a chance to update the status to GitHub.

#### If one of the tests in my PR fails, do I need to run them all again, or just one?

Simply re-run the failed test.

## Reference Documents

- [Maintainer's Guide to Tide](https://github.com/kubernetes/test-infra/blob/master/prow/cmd/tide/maintainers.md)
- [bors-ng](https://github.com/bors-ng/bors-ng)


