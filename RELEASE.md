# TiChi Release Process

## Overview

TiChi uses [goreleaser](https://github.com/goreleaser/goreleaser) to release
in [GitHub Actions](https://github.com/ti-community-infra/tichi/blob/master/.github/workflows/release.yml), where the
task compiles and packages the binary images.

## Release Steps

- Normal Release
    1. Fetch the latest code for upstream: `git fetch upstream`
    2. Create tag: `git tag v1.5.12`
    3. Push tag to upstream: `git push upstream --tags`
    4. Triggering GitHub Actions for releasing

- Release exception, need to re-release
    1. Delete local code tag: `git tag -d v1.5.12`
    2. Delete upstream tag: `git push --delete upstream v1.5.12`
    3. Delete GitHub release drafts (if any)
    4. Fix the issue and re-run the normal releasing process

## Deployment Steps

We use [scripts](https://github.com/ti-community-infra/tichi/blob/master/scripts/deploy.sh) to automate deployment and
upgrades. When a pull request for bump version is merged, we use
a [prow job](https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md)
to run the script to complete the automatic deployment.

These tasks are defined in the following prow jobs:

- [Development Environment Deployment](https://github.com/ti-community-infra/tichi/blob/9f829ae5ba61aaaf149dd046be09262fd3a0e4bc/.prow.yaml#L134)
- [Production Environment Deployment](https://github.com/ti-community-infra/configs/blob/048b5dd46e57fa95c731252d7c95c7cd7a13e1e2/prow/jobs/ti-community-infra/configs/configs-postsubmits.yaml#L35)

In addition, for the test-infra components, we use a bot for
the [automatic bump](https://github.com/kubernetes/test-infra/blob/master/prow/cmd/autobump/autobump.sh), also defined
in prow jobs:

- [Development Environment Bump](https://github.com/ti-community-infra/configs/blob/048b5dd46e57fa95c731252d7c95c7cd7a13e1e2/prow/jobs/ti-community-infra/tichi/tichi-periodics.yaml#L2)
- [Production Environment Bump](https://github.com/ti-community-infra/configs/blob/048b5dd46e57fa95c731252d7c95c7cd7a13e1e2/prow/jobs/ti-community-infra/configs/configs-periodics.yaml#L2)

### Note

**When we bump [test-infra](https://github.com/kubernetes/test-infra) components we need to
check [ANNOUNCEMENTS.md](https://github.com/kubernetes/test-infra/blob/master/prow/ANNOUNCEMENTS.md) to see if the
update has Breaking Changes, and we need to test the development environment and see how the cluster is running before
upgrading the prod environment.**