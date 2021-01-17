module github.com/ti-community-infra/tichi

go 1.15

replace (
	k8s.io/api => k8s.io/api v0.19.3
	k8s.io/client-go => k8s.io/client-go v0.19.3
)

require (
	github.com/gin-gonic/gin v1.6.3
	github.com/google/martian v2.1.1-0.20190517191504-25dcb96d9e51+incompatible
	github.com/kylelemons/godebug v1.1.0
	github.com/shurcooL/githubv4 v0.0.0-20191102174205-af46314aec7b
	github.com/sirupsen/logrus v1.7.0
	gotest.tools v2.2.0+incompatible
	k8s.io/apimachinery v0.19.3
	k8s.io/test-infra v0.0.0-20201223014427-026679a0c7dd
	sigs.k8s.io/yaml v1.2.0
)
