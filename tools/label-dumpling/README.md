# label-dumpling

`label-dumpling` helps us dump all the labels of the repo in [label_sync](https://github.com/kubernetes/test-infra/tree/master/label_sync) format.

## Usage

To use it, you need to set a valid personal access token in the GITHUB_TOKEN environment variable and try the command:

```shell
label-dumpling <org> <repo> --output <output>
```

The output is the path to the output file. The default is `labels.yaml`.
