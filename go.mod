module github.com/networkservicemesh/cmd-nse-supplier-k8s

go 1.16

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/edwarnicke/grpcfd v1.1.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/networkservicemesh/api v1.3.2-0.20220514193644-73abc067b2ce
	github.com/networkservicemesh/sdk v0.5.1-0.20220514221142-f0081c8f06e7
	github.com/networkservicemesh/sdk-k8s v0.0.0-20220514221638-1b04c39e24ab
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spiffe/go-spiffe/v2 v2.0.0
	google.golang.org/grpc v1.42.0
	k8s.io/client-go v0.22.1
)
