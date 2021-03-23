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

package server

import (
	"context"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/rode/collector-build/proto/v1alpha1"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/build_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/provenance_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/source_go_proto"
	"go.uber.org/zap"
)

const (
	rodeProjectId      = "projects/rode"
	buildCollectorNote = rodeProjectId + "/notes/build_collector"
)

type BuildCollectorServer struct {
	logger *zap.Logger
	rode   pb.RodeClient
}

func NewBuildCollectorServer(logger *zap.Logger, rode pb.RodeClient) *BuildCollectorServer {
	return &BuildCollectorServer{
		logger,
		rode,
	}
}

func (s *BuildCollectorServer) CreateBuild(ctx context.Context, request *v1alpha1.CreateBuildRequest) (*v1alpha1.CreateBuildResponse, error) {
	var artifacts []*provenance_go_proto.Artifact

	for _, artifact := range request.Artifacts {
		artifacts = append(artifacts, &provenance_go_proto.Artifact{
			Id: artifact,
		})
	}

	buildOccurrences := &pb.BatchCreateOccurrencesRequest{
		Occurrences: []*grafeas_go_proto.Occurrence{
			{
				Resource: &grafeas_go_proto.Resource{
					Uri: request.Repository,
				},
				NoteName: buildCollectorNote,
				Kind:     common_go_proto.NoteKind_BUILD,
				Details: &grafeas_go_proto.Occurrence_Build{
					Build: &build_go_proto.Details{
						Provenance: &provenance_go_proto.BuildProvenance{
							Id:             uuid.New().String(),
							ProjectId:      rodeProjectId,
							BuiltArtifacts: artifacts,
							CreateTime:     ptypes.TimestampNow(),
							SourceProvenance: &provenance_go_proto.Source{
								Context: &source_go_proto.SourceContext{
									Context: &source_go_proto.SourceContext_Git{
										Git: &source_go_proto.GitSourceContext{
											Url: request.Repository,
										}},
								},
							},
						},
					},
				},
			},
		},
	}

	response, err := s.rode.BatchCreateOccurrences(ctx, buildOccurrences)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.CreateBuildResponse{
		BuildOccurrenceId: response.Occurrences[0].Name,
	}, nil
}
