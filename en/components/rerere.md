# rerere

[rerere](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/rerere) is a core component of tichi, rerere will push the code to a branch dedicated to retesting Re-test on. It will run as a [Prow Job](https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md).

## design background

This component was developed to solve the problem of multiple PR merging mentioned in [Tide](components/tide.md):

- PR1: Rename bifurcate() to bifurcateCrab()
- PR2: call bifurcate()

At this time, both PRs will use the current master as the Base branch for testing, and both PRs will pass. But once PR1 is merged into the master branch first, after the second PR merges (because the test also passed), it will cause the master to fail to find `bifurcate` error.

In order to solve the above problems, we developed [tars](plugins/tars.md) to automatically merge the latest Base branch to PR, which can solve the problem, but it is not an efficient solution for large repositories. **For example, if there are n PRs that can be merged at the same time, then O(n^2) tests must be run, which greatly wastes CI resources. **

In order to efficiently merge PR and save test resources, we propose [multiple solutions](https://github.com/ti-community-infra/rfcs/discussions/13). Finally decided to use Prow Job combined with Tide to solve the problem.

## Design ideas

1. Prow Job will use the [clonerefs](https://github.com/kubernetes/test-infra/tree/master/prow/clonerefs) tool to merge the PR and the latest Base branch and then clone it to the Pod running the test, so We can always get the code base that has merged the latest Base, rerere can push the code to the retest branch for testing before the merge.
2. Porw Job can set `max_concurrency` to control the maximum number of executions of the CI task. This is a natural FIFO queue. We can use this function to queue up retest tasks.
3. Before the code is merged, Tide will check whether all Prow Jobs are tested using the latest Base. If the current CI Pod is not using the latest Base, Tide will automatically re-trigger the test, which ensures that all PRs will be retested with the latest Base before merging.
4. In rerere, we will push the code to the designated test branch, and then regularly check whether all required CIs have passed. When all required CIs have passed, our rerere Prow Job will pass the test. Tide will automatically merge the current PR.

## Parameter configuration

| Parameter name   | Type          | Description                                                                  |
| ---------------- | ------------- | ---------------------------------------------------------------------------- |
| retesting-branch | string        | Branch name for retesting                                                    |
| retry            | int           | The number of retries after the retest timeout                               |
| timeout          | time.Duration | timeout for each retest attempt                                              |
| labels           | []string      | Labels that must be present before retesting (for example: status/can-merge) |
| require-contexts | []string      | The name of the required CI task                                             |


Fox example:

```yaml
presubmits:
  tikv/tikv:
    -name: pull-tikv-master-rerere
      decorate: true
      trigger: "(?mi)^/(merge|rerere)\\s*$"
      rerun_command: "/rerere"
      max_concurrency: 1 # Run at most one task at the same time
      branches:
        -^master$
      spec:
        containers:
          -image: ticommunityinfra/rerere-component:latest
            command:
              - rerere
            args:
              ---github-token-path=/etc/github/token
              ---dry-run=false
              ---github-endpoint=https://api.github.com
              ---retesting-branch=master-retesting
              ---timeout=40m
              ---require-contexts=tikv_prow_integration_common_test/master-retesting
              ---require-contexts=tikv_prow_integration_compatibility_test/master-retesting
              ---require-contexts=tikv_prow_integration_copr_test/master-retesting
              ---require-contexts=tikv_prow_integration_ddl_test/master-retesting
            volumeMounts:
              -name: github-token
                mountPath: /etc/github
                readOnly: true
        volumes:
          -name: github-token
            secret:
              secretName: github-token
```

## Reference documentation

- [Code Implementation](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/rerere)

## Q&A

### Will rerere be triggered every time I submit?

Yes, but we will not actually push to the test branch for testing before we hit `status/can-merge`. The test will be skipped because of the lack of required labels. When the `/merge` command is used, the CI will be triggered again. At this time, because there is already a `status/can-merge`, we will actually re-test.

### If the test fails, do I still need `/merge`?

No, after using Tide, we don't need to use `/merge` continuously, just trigger the test again if a test fails. For example, if the rerere test fails, you only need to re-trigger `/rerere` to make it retest.

### After I merge the PR manually, will it cause problems in the queue for retesting?

No, except for the PR that you manually merged will be affected, other PRs will detect the Base change before the merge, and Tide will automatically re-trigger rerere for testing. 

**It is strongly recommended not to merge PR manually, just re-trigger after the test fails. When all required CIs pass, Tide will try to merge again. **