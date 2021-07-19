module github.com/networkservicemesh/cmd-nse-supplier-k8s

go 1.16

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/edwarnicke/grpcfd v0.1.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/networkservicemesh/api v1.0.1-0.20210715134717-6e4a0f8eae3e
	github.com/networkservicemesh/sdk v0.5.1-0.20210719132747-b086a10c94fe
	github.com/networkservicemesh/sdk-k8s v0.0.0-20210719143312-e20e23e8ea19
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spiffe/go-spiffe/v2 v2.0.0-beta.5
	google.golang.org/grpc v1.37.1
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
)
