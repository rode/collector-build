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
	. "github.com/onsi/gomega"
	"testing"
)

func TestConfig(t *testing.T) {
	Expect := NewGomegaWithT(t).Expect

	for _, tc := range []struct {
		name        string
		flags       []string
		expected    *Config
		expectError bool
	}{
		{
			name:  "defaults",
			flags: []string{},
			expected: &Config{
				GrpcPort: 8082,
				HttpPort: 8083,
				Debug:    false,
				RodeConfig: &RodeConfig{
					Host: "rode:50051",
				},
			},
		},
		{
			name:        "bad grpc port",
			flags:       []string{"--grpc-port=foo"},
			expectError: true,
		},
		{
			name:        "bad http port",
			flags:       []string{"--http-port=bar"},
			expectError: true,
		},
		{
			name:        "bad debug",
			flags:       []string{"--debug=baz"},
			expectError: true,
		},
		{
			name:  "Rode host",
			flags: []string{"--rode-host=bar"},
			expected: &Config{
				GrpcPort: 8082,
				HttpPort: 8083,
				Debug:    false,
				RodeConfig: &RodeConfig{
					Host: "bar",
				},
			},
		},
		{
			name:  "Rode insecure",
			flags: []string{"--rode-insecure=true"},
			expected: &Config{
				GrpcPort: 8082,
				HttpPort: 8083,
				Debug:    false,
				RodeConfig: &RodeConfig{
					Host:     "rode:50051",
					Insecure: true,
				},
			},
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, err := Build("collector-build", tc.flags)

			if tc.expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(c).To(BeEquivalentTo(tc.expected))
			}
		})
	}
}
