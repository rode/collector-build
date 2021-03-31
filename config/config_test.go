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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	rodeconfig "github.com/rode/rode/config"

	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("Build", func() {
		DescribeTable("invalid configuration", func(flags []string) {
			c, err := Build("collector-build", flags)

			Expect(err).To(HaveOccurred())
			Expect(c).To(BeNil())
		},
			Entry("bad grpc port", []string{"--grpc-port=foo"}),
			Entry("bad http port", []string{"--http-port=bar"}),
			Entry("bad debug", []string{"--debug=baz"}),
			Entry("jwt required audience without issuer", []string{"--jwt-required-audience=foo"}),
		)

		DescribeTable("successful configuration", func(flags []string, expected interface{}) {
			c, err := Build("collector-build", flags)

			Expect(err).To(BeNil())
			Expect(c).To(Equal(expected))
		},
			Entry("default config", []string{}, &Config{
				Auth: &rodeconfig.AuthConfig{
					JWT: &rodeconfig.JWTAuthConfig{},
				},
				GrpcPort: 8082,
				HttpPort: 8083,
				Debug:    false,
				RodeConfig: &RodeConfig{
					Host: "rode:50051",
				},
			}),
			Entry("Rode host flag", []string{"--rode-host=bar"}, &Config{
				Auth: &rodeconfig.AuthConfig{
					JWT: &rodeconfig.JWTAuthConfig{},
				},
				GrpcPort: 8082,
				HttpPort: 8083,
				Debug:    false,
				RodeConfig: &RodeConfig{
					Host: "bar",
				},
			}),
			Entry("Rode insecure flag", []string{"--rode-insecure=true"}, &Config{
				Auth: &rodeconfig.AuthConfig{
					JWT: &rodeconfig.JWTAuthConfig{},
				},
				GrpcPort: 8082,
				HttpPort: 8083,
				Debug:    false,
				RodeConfig: &RodeConfig{
					Host:     "rode:50051",
					Insecure: true,
				},
			}),
		)

		Describe("jwt authn configuration", func() {
			type discoveryDocument struct {
				Issuer      string   `json:"issuer"`
				AuthURL     string   `json:"authorization_endpoint"`
				TokenURL    string   `json:"token_endpoint"`
				JWKSURL     string   `json:"jwks_uri"`
				UserInfoURL string   `json:"userinfo_endpoint"`
				Algorithms  []string `json:"id_token_signing_alg_values_supported"`
			}

			issuer := "http://localhost:8080/auth/realms/test"
			oidcWellKnown := "/.well-known/openid-configuration"

			responseBytes, err := json.Marshal(&discoveryDocument{
				Issuer:      issuer,
				AuthURL:     "",
				TokenURL:    "",
				JWKSURL:     "",
				UserInfoURL: "",
				Algorithms:  []string{""},
			})
			Expect(err).ToNot(HaveOccurred())
			responseBody := string(responseBytes)

			var (
				actualError       error
				config            *Config
				discoveryResponse *http.Response
				flags             []string
			)

			BeforeEach(func() {
				httpmock.Activate()
				discoveryResponse = httpmock.NewStringResponse(http.StatusOK, responseBody)
				flags = []string{fmt.Sprintf("--jwt-issuer=%s", issuer)}

				httpmock.RegisterResponder("GET", issuer+oidcWellKnown, func(request *http.Request) (*http.Response, error) {
					return discoveryResponse, nil
				})
			})

			JustBeforeEach(func() {
				config, actualError = Build("collector-build", flags)
			})

			AfterEach(func() {
				httpmock.Deactivate()
			})

			When("the issuer flag is passed", func() {
				It("should be set in the config", func() {
					Expect(actualError).ToNot(HaveOccurred())
					Expect(config.Auth.JWT.Issuer).To(Equal(issuer))
				})
			})

			When("the issuer and required audience flag are passed", func() {
				var (
					expectedAudience string
				)

				BeforeEach(func() {
					expectedAudience = fake.LetterN(10)
					flags = []string{fmt.Sprintf("--jwt-issuer=%s", issuer), fmt.Sprintf("--jwt-required-audience=%s", expectedAudience)}
				})

				It("should set both in the configuration", func() {
					Expect(actualError).ToNot(HaveOccurred())
					Expect(config.Auth.JWT.Issuer).To(Equal(issuer))
					Expect(config.Auth.JWT.RequiredAudience).To(Equal(expectedAudience))
				})
			})

			When("an error occurs retrieving the discovery document", func() {
				BeforeEach(func() {
					discoveryResponse = httpmock.NewStringResponse(http.StatusInternalServerError, "error")
				})

				It("should return the error", func() {
					Expect(actualError).To(HaveOccurred())
				})
			})
		})
	})
})
