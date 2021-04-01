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

package config

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/coreos/go-oidc"
	rodeconfig "github.com/rode/rode/config"
)

type Config struct {
	Auth       *rodeconfig.AuthConfig
	GrpcPort   int
	HttpPort   int
	Debug      bool
	RodeConfig *RodeConfig
}

type RodeConfig struct {
	Host     string
	Insecure bool
}

func Build(name string, args []string) (*Config, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)

	c := &Config{
		Auth: &rodeconfig.AuthConfig{
			Basic: &rodeconfig.BasicAuthConfig{},
			JWT:   &rodeconfig.JWTAuthConfig{},
		},
		RodeConfig: &RodeConfig{},
	}

	flags.IntVar(&c.GrpcPort, "grpc-port", 8082, "the port that the build collector's gRPC service should listen on")
	flags.IntVar(&c.HttpPort, "http-port", 8083, "the port that the build collector's gRPC gateway should listen on")
	flags.BoolVar(&c.Debug, "debug", false, "when set, debug mode will be enabled")
	flags.StringVar(&c.Auth.JWT.Issuer, "jwt-issuer", "", "when set, jwt based auth will be enabled for all endpoints. the provided issuer will be used to fetch the discovery document in order to validate received jwts")
	flags.StringVar(&c.Auth.JWT.RequiredAudience, "jwt-required-audience", "", "when set, if jwt based auth is enabled, this audience must be specified within the `aud` claim of any received jwts")

	flags.StringVar(&c.RodeConfig.Host, "rode-host", "rode:50051", "the host to use to connect to rode")
	flags.BoolVar(&c.RodeConfig.Insecure, "rode-insecure", false, "when set, the connection to rode will not use TLS")

	err := flags.Parse(args)
	if err != nil {
		return nil, err
	}

	if c.Auth.JWT.Issuer != "" {
		provider, err := oidc.NewProvider(context.Background(), c.Auth.JWT.Issuer)
		if err != nil {
			return nil, fmt.Errorf("error initializing oidc provider: %v", err)
		}

		oidcConfig := &oidc.Config{}
		if c.Auth.JWT.RequiredAudience != "" {
			oidcConfig.ClientID = c.Auth.JWT.RequiredAudience
		} else {
			oidcConfig.SkipClientIDCheck = true
		}

		c.Auth.JWT.Verifier = provider.Verifier(oidcConfig)
	} else if c.Auth.JWT.RequiredAudience != "" {
		return nil, errors.New("the --jwt-required-audience flag cannot be specified without --jwt-issuer")
	}

	return c, nil
}
