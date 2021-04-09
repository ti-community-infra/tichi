# assign

## Design Background

Collaborating on a large repository requires assigning PRs or issues to specific collaborators to follow up on, but without write access, you can't assign them directly through the GitHub page.

[assign](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/assign) provides a command that allows the bot to assign collaborators and request reviewers.

## Design

The plugin was designed and developed by the Kubernetes community and provides two commands:

- `/[un]assign @someone hi-rustin`: assign or un-assign Issue/PR to someone and hi-rustin.
- `/[un]cc @someone hi-rustin`: request or un-request someone and hi-rustin to review PR.

Note: If you do not specify a GitHub account after the command, it defaults to yourself.

## Parameter Configuration

No configuration

## Reference documentations

- [command help](https://prow.tidb.io/plugins?repo=ti-community-infra%2Ftichi)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/assign)

## Q&A

### Why do you support usernames that do not start with `@`?

> https://github.com/ti-community-infra/tichi/issues/426

When starting with `@`, GitHub automatically sends an email to the corresponding user. Another notification email will send by the bot when the user has been assigned, or requested to review.
To reduce the number of unnecessary emails, `assign` allows usernames that do not start with `@`.

