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
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/collector-build/mocks"
	"github.com/rode/collector-build/proto/v1alpha1"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/build_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/provenance_go_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Server", func() {
	var (
		ctx        context.Context
		mockCtrl   *gomock.Controller
		rodeClient *mocks.MockRodeClient
		server     *BuildCollectorServer
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		rodeClient = mocks.NewMockRodeClient(mockCtrl)

		server = NewBuildCollectorServer(zap.NewNop(), rodeClient)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("CreateBuild", func() {
		var (
			request     *v1alpha1.CreateBuildRequest
			response    *v1alpha1.CreateBuildResponse
			actualError error
		)

		BeforeEach(func() {
			request = &v1alpha1.CreateBuildRequest{
				Artifacts: []*v1alpha1.Artifact{
					createRandomArtifact(),
					createRandomArtifact(),
				},
				CommitId:     fake.LetterN(10),
				ProvenanceId: fake.Word(),
				LogsUri:      fake.URL(),
				Creator:      fake.Email(),
				BuildStart:   timestamppb.Now(),
				BuildEnd:     timestamppb.New(time.Now().Add(5 * time.Minute)),
				Repository:   fake.URL(),
			}
		})

		JustBeforeEach(func() {
			response, actualError = server.CreateBuild(ctx, request)
		})

		Describe("successful build occurrence creation", func() {
			var (
				expectedOccurrenceId string
				actualRequest        *pb.BatchCreateOccurrencesRequest
			)

			BeforeEach(func() {
				newOccurrence := &grafeas_go_proto.Occurrence{
					Name: "projects/rode/occurrences/" + expectedOccurrenceId,
				}

				request.Repository = "https://github.com/rode/collector-build"

				rodeClient.EXPECT().
					BatchCreateOccurrences(ctx, gomock.Any()).
					Do(func(_ context.Context, r *pb.BatchCreateOccurrencesRequest) {
						actualRequest = r
					}).
					Return(&pb.BatchCreateOccurrencesResponse{Occurrences: []*grafeas_go_proto.Occurrence{newOccurrence}}, nil).
					Times(1)
			})

			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should send a single occurrence", func() {
				Expect(actualRequest.Occurrences).To(HaveLen(1))
			})

			It("should set the resource uri using the repository and commit id", func() {
				expectedResourceUri := "git://github.com/rode/collector-build@" + request.CommitId

				Expect(actualRequest.Occurrences[0].Resource.Uri).To(Equal(expectedResourceUri))
			})

			It("should set a static note name", func() {
				Expect(actualRequest.Occurrences[0].NoteName).To(Equal("projects/rode/notes/build_collector"))
			})

			It("should set the kind as BUILD", func() {
				Expect(actualRequest.Occurrences[0].Kind).To(Equal(common_go_proto.NoteKind_BUILD))
			})

			It("should set the provenance id", func() {
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.Id).To(Equal(request.ProvenanceId))
			})

			It("should set the build creator", func() {
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.Creator).To(Equal(request.Creator))
			})

			It("should set the provenance start, end, and create times", func() {
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.StartTime).To(Equal(request.BuildStart))
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.EndTime).To(Equal(request.BuildEnd))
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.CreateTime.IsValid()).To(Equal(true))
			})

			It("should set the logsUri in the build provenance", func() {
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.LogsUri).To(Equal(request.LogsUri))
			})

			It("should use the Rode project", func() {
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.ProjectId).To(Equal("projects/rode"))
			})

			It("should set the source context", func() {
				source := actualRequest.Occurrences[0].GetBuild().Provenance.SourceProvenance.Context.GetGit()

				Expect(source.Url).To(Equal(request.Repository))
				Expect(source.RevisionId).To(Equal(request.CommitId))
			})

			It("should include the specified artifacts", func() {
				var expectedArtifacts []*provenance_go_proto.Artifact
				for _, a := range request.Artifacts {
					expectedArtifacts = append(expectedArtifacts, &provenance_go_proto.Artifact{
						Id:    a.Id,
						Names: a.Names,
					})
				}
				actualArtifacts := actualRequest.Occurrences[0].GetBuild().Provenance.BuiltArtifacts

				Expect(actualArtifacts).To(ConsistOf(expectedArtifacts))
			})

			It("should return the occurrence id", func() {
				Expect(response.BuildOccurrenceId).To(Equal(expectedOccurrenceId))
			})

			Describe("build start is not specified", func() {
				BeforeEach(func() {
					request.BuildStart = nil
				})

				It("should use the current time", func() {
					actualStartTime := actualRequest.Occurrences[0].GetBuild().Provenance.StartTime
					Expect(actualStartTime.IsValid()).To(BeTrue())
				})
			})

			Describe("build end is not specified", func() {
				BeforeEach(func() {
					request.BuildEnd = nil
				})

				It("should use the current time", func() {
					actualEndTime := actualRequest.Occurrences[0].GetBuild().Provenance.EndTime
					Expect(actualEndTime.IsValid()).To(BeTrue())
				})
			})
		})

		Describe("request validation", func() {
			When("the request is missing the repository", func() {
				BeforeEach(func() {
					request.Repository = ""
				})

				It("should return an error", func() {
					Expect(response).To(BeNil())
					Expect(actualError).To(HaveOccurred())
				})

				It("should set the status to invalid argument", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(codes.InvalidArgument))
					Expect(s.Message()).To(Equal("Invalid request: no repository specified"))
				})
			})

			When("the request contains an invalid repository url", func() {
				BeforeEach(func() {
					request.Repository = fake.Word()
				})

				It("should return an error", func() {
					Expect(response).To(BeNil())
					Expect(actualError).To(HaveOccurred())
				})

				It("should set the status to invalid argument", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(codes.InvalidArgument))
					Expect(s.Message()).To(ContainSubstring("Invalid repository url"))
				})
			})

			When("the request contains no artifacts", func() {
				BeforeEach(func() {
					request.Artifacts = []*v1alpha1.Artifact{}
				})

				It("should return an error", func() {
					Expect(response).To(BeNil())
					Expect(actualError).To(HaveOccurred())
				})

				It("should set the status to invalid argument", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(codes.InvalidArgument))
					Expect(s.Message()).To(Equal("Invalid request: no artifacts specified"))
				})
			})

			When("the request commit id is empty", func() {
				BeforeEach(func() {
					request.CommitId = ""
				})

				It("should return an error", func() {
					Expect(response).To(BeNil())
					Expect(actualError).To(HaveOccurred())
				})

				It("should set the status to invalid argument", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(codes.InvalidArgument))
					Expect(s.Message()).To(Equal("Invalid request: no commit ID specified"))
				})
			})
		})

		Describe("error occurs while creating occurrence", func() {
			var (
				expectedError error
			)

			BeforeEach(func() {
				expectedError = fmt.Errorf(fake.Word())
				rodeClient.EXPECT().
					BatchCreateOccurrences(gomock.Any(), gomock.Any()).
					Return(nil, expectedError).
					Times(1)
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(response).To(BeNil())
			})

			It("should set the status to internal server error", func() {
				s := getGRPCStatusFromError(actualError)

				Expect(s.Code()).To(Equal(codes.Internal))
				Expect(s.Message()).To(Equal(fmt.Sprintf("Error creating occurrences in Rode: %s", expectedError)))
			})
		})

		Describe("BatchCreateOccurrences does not return expected occurrence", func() {
			BeforeEach(func() {
				rodeClient.EXPECT().
					BatchCreateOccurrences(gomock.Any(), gomock.Any()).
					Return(&pb.BatchCreateOccurrencesResponse{}, nil).
					Times(1)
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(response).To(BeNil())
			})

			It("should set the status to internal server error", func() {
				s := getGRPCStatusFromError(actualError)

				Expect(s.Code()).To(Equal(codes.Internal))
				Expect(s.Message()).To(Equal(fmt.Sprintf("Occurrence data not returned from Rode")))
			})
		})
	})

	Describe("UpdateBuildArtifacts", func() {
		var (
			expectedOccurrenceId    string
			request                 *v1alpha1.UpdateBuildArtifactsRequest
			expectedOccurrence      *grafeas_go_proto.Occurrence
			listOccurrencesResponse *pb.ListOccurrencesResponse

			actualError    error
			actualResponse *v1alpha1.UpdateBuildArtifactsResponse
		)

		BeforeEach(func() {
			expectedOccurrenceId = fake.UUID()
			request = &v1alpha1.UpdateBuildArtifactsRequest{
				ExistingArtifactId: fake.URL(),
				NewArtifact:        createRandomArtifact(),
			}

			expectedOccurrence = makeBuildOccurrence(expectedOccurrenceId, request.ExistingArtifactId)

			listOccurrencesResponse = &pb.ListOccurrencesResponse{
				Occurrences: []*grafeas_go_proto.Occurrence{expectedOccurrence},
			}
		})

		JustBeforeEach(func() {
			actualResponse, actualError = server.UpdateBuildArtifacts(ctx, request)
		})

		Describe("successful occurrence update", func() {
			var (
				actualListOccurrencesRequest  *pb.ListOccurrencesRequest
				actualUpdateOccurrenceRequest *pb.UpdateOccurrenceRequest
			)

			BeforeEach(func() {
				rodeClient.EXPECT().
					UpdateOccurrence(ctx, gomock.Any()).
					Do(func(_ context.Context, req *pb.UpdateOccurrenceRequest) {
						actualUpdateOccurrenceRequest = req
					}).Return(expectedOccurrence, nil)

				rodeClient.EXPECT().
					ListOccurrences(ctx, gomock.Any()).
					Do(func(_ context.Context, req *pb.ListOccurrencesRequest) {
						actualListOccurrencesRequest = req
					}).
					Return(listOccurrencesResponse, nil)
			})

			When("there's a single matching occurrence to update", func() {
				It("should filter all occurrences using the existing artifact", func() {
					expectedFilter := fmt.Sprintf(`build.provenance.builtArtifacts.nestedFilter(id == "%s")`, request.ExistingArtifactId)

					Expect(actualListOccurrencesRequest.Filter).To(Equal(expectedFilter))
				})

				It("should append the new artifact", func() {
					actualArtifacts := actualUpdateOccurrenceRequest.Occurrence.GetBuild().Provenance.BuiltArtifacts

					Expect(actualArtifacts).To(ConsistOf(
						&provenance_go_proto.Artifact{
							Id: request.ExistingArtifactId,
						},
						&provenance_go_proto.Artifact{
							Id:    request.NewArtifact.Id,
							Names: request.NewArtifact.Names,
						},
					))
				})

				It("should set the update mask to only change the artifacts", func() {
					actualMask := actualUpdateOccurrenceRequest.UpdateMask

					Expect(actualMask.Paths).To(ConsistOf("details.build.provenance.built_artifacts"))
				})

				It("should return the occurrence id", func() {
					Expect(actualResponse.BuildOccurrenceId).To(Equal(expectedOccurrenceId))
				})
			})

			When("there are multiple occurrences tied to an artifact", func() {
				var (
					newestBuildOccurrence *grafeas_go_proto.Occurrence
					oldestBuildOccurrence *grafeas_go_proto.Occurrence
				)

				BeforeEach(func() {
					createTime := time.Now()

					oldestBuildOccurrence = makeBuildOccurrence(expectedOccurrenceId, request.ExistingArtifactId)
					oldestBuildOccurrence.CreateTime = timestamppb.New(createTime)

					newestBuildOccurrence = makeBuildOccurrence(fake.UUID(), request.ExistingArtifactId)
					newestBuildOccurrence.CreateTime = timestamppb.New(createTime.Add(time.Minute * 5))

					listOccurrencesResponse.Occurrences = []*grafeas_go_proto.Occurrence{
						newestBuildOccurrence,
						oldestBuildOccurrence,
					}
				})

				It("should update the oldest build occurrence", func() {
					Expect(actualUpdateOccurrenceRequest.Id).To(Equal(expectedOccurrenceId))
				})
			})
		})

		Describe("updating the occurrence is unsuccessful", func() {
			var (
				expectedError error
			)

			BeforeEach(func() {
				expectedError = errors.New(fake.Word())
			})

			Describe("request validation", func() {
				When("request is missing the existing artifact slug", func() {
					BeforeEach(func() {
						request.ExistingArtifactId = ""
					})

					It("should return an error", func() {
						Expect(actualError).To(HaveOccurred())
						Expect(actualResponse).To(BeNil())
					})

					It("should set the status to invalid argument", func() {
						s := getGRPCStatusFromError(actualError)

						Expect(s.Code()).To(Equal(codes.InvalidArgument))
						Expect(s.Message()).To(Equal("Invalid request: existing artifact must be specified"))
					})
				})

				When("request is missing the new artifact slug", func() {
					BeforeEach(func() {
						request.NewArtifact = nil
					})

					It("should return an error", func() {
						Expect(actualError).To(HaveOccurred())
						Expect(actualResponse).To(BeNil())
					})

					It("should set the status to invalid argument", func() {
						s := getGRPCStatusFromError(actualError)

						Expect(s.Code()).To(Equal(codes.InvalidArgument))
						Expect(s.Message()).To(Equal("Invalid request: new artifact must be specified"))
					})
				})
			})

			When("an error occurs listing occurrences", func() {
				BeforeEach(func() {
					rodeClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(nil, expectedError)
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})

				It("should set the status to internal server error", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(codes.Internal))
					Expect(s.Message()).To(ContainSubstring(expectedError.Error()))
				})
			})

			When("no occurrences have matching artifacts", func() {
				BeforeEach(func() {
					listOccurrencesResponse.Occurrences = []*grafeas_go_proto.Occurrence{}

					rodeClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(listOccurrencesResponse, nil)
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})

				It("should set the status to invalid argument", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(codes.NotFound))
					Expect(s.Message()).To(ContainSubstring("No occurrence found for artifact"))
				})
			})

			When("the call to UpdateOccurrence fails", func() {
				BeforeEach(func() {
					rodeClient.EXPECT().ListOccurrences(gomock.Any(), gomock.Any()).Return(listOccurrencesResponse, nil)
					rodeClient.EXPECT().UpdateOccurrence(gomock.Any(), gomock.Any()).Return(nil, expectedError)
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})

				It("should set the status to internal server error", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(codes.Internal))
					Expect(s.Message()).To(ContainSubstring("Error updating existing artifact in Rode"))
				})
			})
		})
	})
})

func getGRPCStatusFromError(err error) *status.Status {
	s, ok := status.FromError(err)
	Expect(ok).To(BeTrue(), "Expected error to be a gRPC status")

	return s
}

func makeBuildOccurrence(occurrenceId, artifact string) *grafeas_go_proto.Occurrence {
	return &grafeas_go_proto.Occurrence{
		Name: fmt.Sprintf("projects/rode/occurrences/%s", occurrenceId),
		Details: &grafeas_go_proto.Occurrence_Build{
			Build: &build_go_proto.Details{
				Provenance: &provenance_go_proto.BuildProvenance{
					BuiltArtifacts: []*provenance_go_proto.Artifact{
						{
							Id: artifact,
						},
					},
				},
			},
		},
	}
}

func createRandomArtifact() *v1alpha1.Artifact {
	return &v1alpha1.Artifact{
		Id: fake.URL(),
		Names: []string{
			fake.URL(),
			fake.URL(),
		},
	}
}
