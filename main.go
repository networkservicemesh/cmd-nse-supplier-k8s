// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !windows
// +build !windows

package main

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/edwarnicke/grpcfd"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk-k8s/pkg/networkservice/common/createpod"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/networkservicemesh/sdk/pkg/tools/debug"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
)

// Config holds configuration parameters from environment variables
type Config struct {
	Name                  string            `default:"nse-supplier-k8s" desc:"Name of the Server" split_words:"true"`
	ConnectTo             url.URL           `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"url to connect to" split_words:"true"`
	MaxTokenLifetime      time.Duration     `default:"10m" desc:"maximum lifetime of tokens" split_words:"true"`
	ServiceName           string            `default:"nse-supplier-k8s" desc:"Name of providing service" split_words:"true"`
	Payload               string            `default:"ETHERNET" desc:"Name of provided service payload" split_words:"true"`
	Labels                map[string]string `default:"" desc:"Endpoint labels" split_words:"true"`
	PodDescriptionFile    string            `default:"pod.yaml" desc:"Path to the file that describes pod to be created" split_words:"true"`
	Namespace             string            `default:"default" desc:"Namespace in which new pods will be created" split_words:"true"`
	LogLevel              string            `default:"INFO" desc:"Log level" split_words:"true"`
	OpenTelemetryEndpoint string            `default:"otel-collector.observability.svc.cluster.local:4317" desc:"OpenTelemetry Collector Endpoint"`
}

// Process prints and processes env to config
func (c *Config) Process() error {
	if err := envconfig.Usage("nsm", c); err != nil {
		return errors.Wrap(err, "cannot show usage of envconfig nse")
	}
	if err := envconfig.Process("nsm", c); err != nil {
		return errors.Wrap(err, "cannot process envconfig nse")
	}
	return nil
}

func main() {
	// ********************************************************************************
	// setup context to catch signals
	// ********************************************************************************
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		// More Linux signals here
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	// ********************************************************************************
	// setup logging
	// ********************************************************************************
	log.EnableTracing(true)
	logrus.SetFormatter(&nested.Formatter{})
	ctx = log.WithLog(ctx, logruslogger.New(ctx, map[string]interface{}{"cmd": os.Args[0]}))

	logger := log.FromContext(ctx)

	if err := debug.Self(); err != nil {
		logger.Infof("%s", err)
	}

	// enumerating phases
	logger.Infof("there are 6 phases which will be executed followed by a success message:")
	logger.Infof("the phases include:")
	logger.Infof("1: get config from environment")
	logger.Infof("2: retrieve spiffe svid")
	logger.Infof("3: get kubernetes config")
	logger.Infof("4: create supplier endpoint")
	logger.Infof("5: create grpc and mount nse")
	logger.Infof("6: register nse with nsm")
	logger.Infof("a final success message with start time duration")

	starttime := time.Now()

	// ********************************************************************************
	logger.Infof("executing phase 1: get config from environment")
	// ********************************************************************************
	config := new(Config)
	if err := config.Process(); err != nil {
		logger.Fatal(err.Error())
	}
	l, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logrus.Fatalf("invalid log level %s", config.LogLevel)
	}
	logrus.SetLevel(l)
	logger.Infof("Config: %#v", config)

	// ********************************************************************************
	// Configure Open Telemetry
	// ********************************************************************************
	if opentelemetry.IsEnabled() {
		collectorAddress := config.OpenTelemetryEndpoint
		spanExporter := opentelemetry.InitSpanExporter(ctx, collectorAddress)
		metricExporter := opentelemetry.InitMetricExporter(ctx, collectorAddress)
		o := opentelemetry.Init(ctx, spanExporter, metricExporter, config.Name)
		defer func() {
			if err = o.Close(); err != nil {
				logger.Error(err.Error())
			}
		}()
	}

	// ********************************************************************************
	logger.Infof("executing phase 2: retrieving svid, check spire agent logs if this is the last line you see")
	// ********************************************************************************
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		logger.Fatalf("error getting x509 source: %+v", err)
	}
	svid, err := source.GetX509SVID()
	if err != nil {
		logger.Fatalf("error getting x509 svid: %+v", err)
	}
	logger.Infof("SVID: %q", svid.ID)

	// ********************************************************************************
	logger.Infof("executing phase 3: getting kubernetes config and pod description")
	// ********************************************************************************
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalf("can't get kuberneted config. Are you running this app inside kuberneted pod?")
	}
	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		logger.Fatalf("can't get kubernetes client")
	}
	logger.Infof("successfully obtained kubernetes client")

	podYamlBytes, err := ioutil.ReadFile(config.PodDescriptionFile)
	if err != nil {
		logger.Fatalf("can't read pod file: %+v", err)
	}

	// ********************************************************************************
	logger.Infof("executing phase 4: create supplier endpoint")
	// ********************************************************************************
	supplierEndpoint := endpoint.NewServer(ctx,
		spiffejwt.TokenGeneratorFunc(source, config.MaxTokenLifetime),
		endpoint.WithName(config.Name),
		endpoint.WithAuthorizeServer(authorize.NewServer()),
		endpoint.WithAdditionalFunctionality(
			createpod.NewServer(ctx, client, string(podYamlBytes), createpod.WithNamespace(config.Namespace)),
		),
	)

	// ********************************************************************************
	logger.Infof("executing phase 5: create grpc server and register the server")
	// ********************************************************************************
	options := append(
		tracing.WithTracing(),
		grpc.Creds(
			grpcfd.TransportCredentials(
				credentials.NewTLS(
					tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny()),
				),
			),
		),
	)
	server := grpc.NewServer(options...)
	supplierEndpoint.Register(server)
	tmpDir, err := ioutil.TempDir("", config.Name)
	if err != nil {
		logger.Fatalf("error creating tmpDir %+v", err)
	}
	defer func(tmpDir string) { _ = os.Remove(tmpDir) }(tmpDir)
	listenOn := &(url.URL{Scheme: "unix", Path: filepath.Join(tmpDir, "listen.on")})
	srvErrCh := grpcutils.ListenAndServe(ctx, listenOn, server)
	exitOnErr(ctx, cancel, srvErrCh)
	logger.Infof("grpc server started")

	// ********************************************************************************
	logger.Infof("executing phase 6: register nse with nsm")
	// ********************************************************************************
	clientOptions := append(
		tracing.WithTracingDial(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(
				credentials.NewTLS(
					tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny()),
				),
			),
		),
	)

	nseRegistryClient := registryclient.NewNetworkServiceEndpointRegistryClient(
		ctx,
		registryclient.WithClientURL(&config.ConnectTo),
		registryclient.WithDialOptions(clientOptions...),
		registryclient.WithNSEAdditionalFunctionality(sendfd.NewNetworkServiceEndpointRegistryClient()),
	)

	nse, err := nseRegistryClient.Register(ctx, &registryapi.NetworkServiceEndpoint{
		Name:                config.Name,
		NetworkServiceNames: []string{config.ServiceName},
		NetworkServiceLabels: map[string]*registryapi.NetworkServiceLabels{
			config.ServiceName: {
				Labels: config.Labels,
			},
		},
		Url: listenOn.String(),
	})
	if err != nil {
		logger.Fatalf("unable to register nse %+v", err)
	}
	logger.Infof("nse: %+v", nse)

	// ********************************************************************************
	logger.Infof("startup completed in %v", time.Since(starttime))
	// ********************************************************************************

	// wait for server to exit
	<-ctx.Done()
}

func exitOnErr(ctx context.Context, cancel context.CancelFunc, errCh <-chan error) {
	// If we already have an error, log it and exit
	select {
	case err := <-errCh:
		log.FromContext(ctx).Fatal(err)
	default:
	}
	// Otherwise wait for an error in the background to log and cancel
	go func(ctx context.Context, errCh <-chan error) {
		err := <-errCh
		log.FromContext(ctx).Error(err)
		cancel()
	}(ctx, errCh)
}
