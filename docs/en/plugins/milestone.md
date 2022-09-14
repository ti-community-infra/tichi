# milestone

## Design Background

On large repositories we use milestones to track the progress of PRs and Issues, but GitHub restricts the ability to add milestones to Issues/PRs to collaborators with write access.

[milestone](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/milestone) provides a command that allows the bot to add the corresponding milestone.

## Permissions Design

This plugin is primarily responsible for adding milestones, so only the milestone management team can use this command.

## Design

The plugin was designed and developed by the Kubernetes community and provides two commands.

- `/milestone v1.3.2 v1.4` adds milestone v1.3.2 and v1.4.
- `/milestone clear` clears all milestones on Issue/PR.

Note: Only the milestone management team can use this command.

## Parameter Configuration

| Parameter Name            | Type   | Description      |
| ------------------------- | ------ | ---------------- |
| maintainers_id            | int    | GitHub Team ID   |
| maintainers_team          | string | GitHub team name |
| maintainers_friendly_name | string | Team nickname    |

For example:

```yaml
repo_milestone:
  ti-community-infra/test-dev:
    maintainers_id: 4300209
    maintainers_team: bots-maintainers
    maintainers_friendly_name: Robots Maintainers
```

## Reference documents

- [command help](https://prow.tidb.net/plugins?repo=ti-community-infra%2Ftichi)
- [code](https://github.com/kubernetes/test-infra/tree/master/prow/plugins/milestone)

## Q&A

### How can I get my GitHub Team ID?

```sh
curl -H "Authorization: token <token>" "https://api.github.com/orgs/<org-name>/teams?page=N"
```

The above API allows you to get the details of all GitHub teams under that organization.
