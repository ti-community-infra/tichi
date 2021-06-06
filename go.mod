module github.com/ti-community-infra/tichi

go 1.16

replace k8s.io/client-go => k8s.io/client-go v0.20.2

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/mroth/weightedrand v0.4.1
	github.com/shurcooL/githubv4 v0.0.0-20191102174205-af46314aec7b
	github.com/sirupsen/logrus v1.7.0
	gotest.tools v2.2.0+incompatible
	k8s.io/apimachinery v0.20.2
	k8s.io/test-infra v0.0.0-20210605052838-aa44f2be7bbc
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/yaml v1.2.0
)
