// Copyright 2021 The Rode Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rode/collector-build/proto/v1alpha1"
	"github.com/rode/collector-build/server"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/soheilhy/cmux"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/rode/collector-build/config"
)

type proxyAuth struct {
	requireTransportSecurity bool
}

func (p *proxyAuth) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	authzHeader := metautils.ExtractIncoming(ctx).Get("authorization")
	metadata := map[string]string{}
	if authzHeader != "" {
		metadata["authorization"] = authzHeader
	}

	return metadata, nil
}

func (p *proxyAuth) RequireTransportSecurity() bool {
	return p.requireTransportSecurity
}

func main() {
	conf, err := config.Build(os.Args[0], os.Args[1:])
	if err != nil {
		log.Fatalf("error parsing flags: %v", err)
	}

	logger, err := createLogger(conf.Debug)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", conf.Port))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	dialOptions := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(&proxyAuth{requireTransportSecurity: !conf.RodeConfig.Insecure}),
	}
	if conf.RodeConfig.Insecure {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	} else {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	conn, err := grpc.DialContext(ctx, conf.RodeConfig.Host, dialOptions...)
	if err != nil {
		logger.Fatal("failed to establish grpc connection to Rode", zap.Error(err))
	}
	defer conn.Close()

	rodeClient := pb.NewRodeClient(conn)
	grpcServer := grpc.NewServer()

	if conf.Debug {
		reflection.Register(grpcServer)
	}

	buildCollectorServer := server.NewBuildCollectorServer(logger, rodeClient)
	v1alpha1.RegisterBuildCollectorServer(grpcServer, buildCollectorServer)

	healthzServer := server.NewHealthzServer(logger.Named("healthz"))
	grpc_health_v1.RegisterHealthServer(grpcServer, healthzServer)

	mux := cmux.New(lis)
	grpcListener := mux.Match(cmux.HTTP2())
	httpListener := mux.Match(cmux.HTTP1())

	grpcGateway, err := createGrpcGateway(context.Background(), lis.Addr().String())
	if err != nil {
		logger.Fatal("failed to create gRPC gateway", zap.Error(err))
	}

	httpMux := http.NewServeMux()
	httpMux.Handle("/", grpcGateway)

	httpServer := &http.Server{
		Handler: httpMux,
	}

	servers := new(errgroup.Group)
	servers.Go(func() error {
		return grpcServer.Serve(grpcListener)
	})
	servers.Go(func() error {
		return httpServer.Serve(httpListener)
	})
	servers.Go(func() error {
		return mux.Serve()
	})

	logger.Info("listening", zap.String("host", lis.Addr().String()))
	healthzServer.Ready()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	terminationSignal := <-sig

	logger.Info("shutting down...", zap.String("termination signal", terminationSignal.String()))
	healthzServer.NotReady()

	grpcServer.GracefulStop()
	httpServer.Shutdown(context.Background())
}

func createGrpcGateway(ctx context.Context, grpcAddress string) (http.Handler, error) {
	conn, err := grpc.DialContext(
		context.Background(),
		grpcAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}
	gwmux := runtime.NewServeMux()
	if err := v1alpha1.RegisterBuildCollectorHandler(ctx, gwmux, conn); err != nil {
		return nil, err
	}

	return http.Handler(gwmux), nil
}

func createLogger(debug bool) (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}
