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

### Why does the bot tell me that I can't `/assign` an issue or PR to someone? Even if the user is a member of the organization, they can't be `/assigned`?

GitHub has some restrictions on assigning:

- Each Issue or PR can only be assigned to a maximum of 10 users.
- The following four types of users can be assigned:
    - Issue or PR Author
    - Users who have comments on the Issue or PR
    - Users with write access to this repository
    - Organization members who have read access to the repository (**Note: public repository visibility is not the same
      as collaborator read access, in this case collaborators who are explicitly added as having read access in the
      repository permission settings**)

Also see the
GitHub [documentation](https://docs.github.com/en/issues/tracking-your-work-with-issues/managing-issues/assigning-issues-and-pull-requests-to-other-github-users)
for the assigning.



