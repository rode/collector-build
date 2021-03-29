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
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

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
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	rodeProjectId                 = "projects/rode"
	buildCollectorNote            = rodeProjectId + "/notes/build_collector"
	buildOccurrenceArtifactFilter = `build.provenance.builtArtifacts.nestedFilter(id == "%s")`
)

var (
	newUuid = uuid.New
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
	log := s.logger.Named("CreateBuild")

	log.Debug("Received request", zap.Any("request", request))

	if err := validateCreateBuildRequest(request); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	buildOccurrence, err := mapRequestToBuildOccurrence(log, request)
	if err != nil {
		return nil, err
	}

	buildOccurrences := &pb.BatchCreateOccurrencesRequest{
		Occurrences: []*grafeas_go_proto.Occurrence{buildOccurrence},
	}

	log.Debug("Calling BatchCreateOccurrences")
	response, err := s.rode.BatchCreateOccurrences(ctx, buildOccurrences)
	if err != nil {
		log.Error("Error occurred when calling BatchCreateOccurrences", zap.Error(err))

		return nil, status.Errorf(codes.Internal, "Error creating occurrences in Rode: %s", err)
	}

	if len(response.Occurrences) != 1 {
		log.Warn("Did not get expected occurrences from Rode", zap.Any("response", response))
		return nil, status.Error(codes.Internal, "Occurrence data not returned from Rode")
	}

	newOccurrence := response.Occurrences[0]

	return &v1alpha1.CreateBuildResponse{
		BuildOccurrenceId: extractOccurrenceIdFromName(newOccurrence.Name),
	}, nil
}

func (s *BuildCollectorServer) UpdateBuildArtifacts(ctx context.Context, request *v1alpha1.UpdateBuildArtifactsRequest) (*v1alpha1.UpdateBuildArtifactsResponse, error) {
	log := s.logger.Named("UpdateBuildArtifacts").With(zap.String("existingArtifact", request.ExistingArtifact), zap.String("newArtifact", request.NewArtifact))
	log.Debug("Received request")

	if err := validateUpdateBuildArtifactsRequest(request); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	artifactFilter := fmt.Sprintf(buildOccurrenceArtifactFilter, request.ExistingArtifact)

	response, err := s.rode.ListOccurrences(ctx, &pb.ListOccurrencesRequest{Filter: artifactFilter})
	if err != nil {
		log.Error("Error occurred when calling ListOccurrences", zap.Error(err))

		return nil, status.Errorf(codes.Internal, "Error finding existing artifact in Rode: %s", err)
	}
	log.Debug("ListOccurrences response", zap.Any("response", response))

	if len(response.Occurrences) == 0 {
		log.Error("No occurrence found for artifact")
		return nil, status.Errorf(codes.NotFound, "No occurrence found for artifact: %s", request.ExistingArtifact)
	}

	if len(response.Occurrences) > 1 {
		log.Warn("More than one occurrence found for artifact, taking earliest")
	}

	sort.Slice(response.Occurrences, func(l, r int) bool {
		left := response.Occurrences[l].CreateTime.AsTime()
		right := response.Occurrences[r].CreateTime.AsTime()

		return left.Before(right)
	})
	occurrence := response.Occurrences[0]

	occurrence.GetBuild().Provenance.BuiltArtifacts = append(
		occurrence.GetBuild().Provenance.BuiltArtifacts,
		&provenance_go_proto.Artifact{
			Id: request.NewArtifact,
		},
	)

	res, err := s.rode.UpdateOccurrence(ctx, &pb.UpdateOccurrenceRequest{
		Id:         extractOccurrenceIdFromName(occurrence.Name),
		Occurrence: occurrence,
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"details.build.provenance.built_artifacts"},
		}})

	log.Debug("UpdateOccurrence response", zap.Any("response", res))

	if err != nil {
		log.Error("Error calling UpdateOccurrence", zap.Error(err))

		return nil, status.Errorf(codes.Internal, "Error updating existing artifact in Rode: %s", err)
	}

	return &v1alpha1.UpdateBuildArtifactsResponse{
		BuildOccurrenceId: extractOccurrenceIdFromName(res.Name),
	}, nil
}

func validateUpdateBuildArtifactsRequest(request *v1alpha1.UpdateBuildArtifactsRequest) error {
	if len(request.NewArtifact) == 0 {
		return errors.New("new artifact must be specified")
	}

	if len(request.ExistingArtifact) == 0 {
		return errors.New("existing artifact must be specified")
	}

	return nil
}

func validateCreateBuildRequest(request *v1alpha1.CreateBuildRequest) error {
	if len(request.Repository) == 0 {
		return errors.New("no repository specified")
	}

	if len(request.Artifacts) == 0 {
		return errors.New("no artifacts specified")
	}

	if len(request.CommitId) == 0 {
		return errors.New("no commit ID specified")
	}

	return nil
}

func mapRequestToBuildOccurrence(log *zap.Logger, request *v1alpha1.CreateBuildRequest) (*grafeas_go_proto.Occurrence, error) {
	repositoryURL, err := url.ParseRequestURI(request.Repository)
	if err != nil {
		log.Error("Invalid repository url", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "Invalid repository url: %s", err)
	}

	var artifacts []*provenance_go_proto.Artifact
	for _, artifact := range request.Artifacts {
		artifacts = append(artifacts, &provenance_go_proto.Artifact{
			Id: artifact,
		})
	}

	return &grafeas_go_proto.Occurrence{
		Resource: &grafeas_go_proto.Resource{
			Uri: fmt.Sprintf("git://%s%s@%s", repositoryURL.Host, repositoryURL.Path, request.CommitId),
		},
		NoteName: buildCollectorNote,
		Kind:     common_go_proto.NoteKind_BUILD,
		Details: &grafeas_go_proto.Occurrence_Build{
			Build: &build_go_proto.Details{
				Provenance: &provenance_go_proto.BuildProvenance{
					Id:             newUuid().String(),
					ProjectId:      rodeProjectId,
					BuiltArtifacts: artifacts,
					CreateTime:     ptypes.TimestampNow(),
					SourceProvenance: &provenance_go_proto.Source{
						Context: &source_go_proto.SourceContext{
							Context: &source_go_proto.SourceContext_Git{
								Git: &source_go_proto.GitSourceContext{
									Url:        request.Repository,
									RevisionId: request.CommitId,
								}},
						},
					},
				},
			},
		},
	}, nil
}

func extractOccurrenceIdFromName(occurrenceName string) string {
	namePieces := strings.Split(occurrenceName, "/")

	return namePieces[len(namePieces)-1]
}
