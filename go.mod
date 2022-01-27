module github.com/networkservicemesh/cmd-nse-supplier-k8s

go 1.16

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/edwarnicke/grpcfd v0.1.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/networkservicemesh/api v1.1.2-0.20220119092736-21eda250c390
	github.com/networkservicemesh/sdk v0.5.1-0.20220127193211-6d5254af4bb4
	github.com/networkservicemesh/sdk-k8s v0.0.0-20220127195208-8e0f9d7f7ac0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spiffe/go-spiffe/v2 v2.0.0-beta.5
	google.golang.org/grpc v1.42.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
)
