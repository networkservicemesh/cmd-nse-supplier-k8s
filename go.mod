module github.com/networkservicemesh/cmd-nse-supplier-k8s

go 1.16

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/edwarnicke/grpcfd v0.1.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/networkservicemesh/api v1.0.1-0.20210811070028-10403c0f20c8
	github.com/networkservicemesh/sdk v0.5.1-0.20210827110808-c2178d55e7a9
	github.com/networkservicemesh/sdk-k8s v0.0.0-20210830070832-6d0b382b100a
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spiffe/go-spiffe/v2 v2.0.0-beta.5
	google.golang.org/grpc v1.37.1
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
)
