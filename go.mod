module github.com/tidb-community-bots/ti-community-prow

go 1.15

replace (
	k8s.io/api => k8s.io/api v0.19.3
	k8s.io/client-go => k8s.io/client-go v0.19.3
)

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/sirupsen/logrus v1.7.0
	github.com/tidb-community-bots/prow-github v1.2.4
	gotest.tools v2.2.0+incompatible
	k8s.io/apimachinery v0.19.3
	k8s.io/test-infra v0.0.0-20201113003225-7913a618fee0
	sigs.k8s.io/yaml v1.2.0
)
