# Prow Introduction

## About Prow

[Prow](https://github.com/kubernetes/test-infra/tree/master/prow) is a Kubernetes based CI/CD system. 
Jobs can be triggered by various types of events and report their status to many services. In addition to job execution, Prow provides GitHub automation in the form of policy enforcement, chat-ops via /foo style commands, and automatic PR merging.

## Prow in TiDB community

In [tichi](https://github.com/ti-community-infra/tichi), the focus is on automating and standardizing the [TiDB](https://github.com/pingcap/tidb) community's collaboration process using the GitHub automation features provided by Prow.

Because Prow is so [extensible](https://github.com/kubernetes/test-infra/tree/master/prow/plugins), it is possible to write and customize [plugins](https://github.com/ti-community-infra/tichi/tree/master/internal/pkg/externalplugins) according to your own community practice specifications.
The TiDB community has customized a number of its own plugins based on this feature.

## About the book

The purpose of this book is to serve as a handbook for collaborators in the TiDB community, and as an example of how to use Prow for the rest of the community.

