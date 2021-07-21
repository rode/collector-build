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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/collector-build/proto/v1alpha1"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/proto/v1alpha1fakes"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/build_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/provenance_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Server", func() {
	var (
		ctx        context.Context
		rodeClient *v1alpha1fakes.FakeRodeClient
		server     *BuildCollectorServer
	)

	BeforeEach(func() {
		ctx = context.Background()
		rodeClient = &v1alpha1fakes.FakeRodeClient{}

		server = NewBuildCollectorServer(logger, rodeClient)
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
				CommitUri:    fake.URL(),
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
			)

			BeforeEach(func() {
				newOccurrence := &grafeas_go_proto.Occurrence{
					Name: "projects/rode/occurrences/" + expectedOccurrenceId,
				}

				request.Repository = "https://github.com/rode/collector-build"

				batchResponse := &pb.BatchCreateOccurrencesResponse{Occurrences: []*grafeas_go_proto.Occurrence{newOccurrence}}
				rodeClient.BatchCreateOccurrencesReturns(batchResponse, nil)
			})

			It("should not return an error", func() {
				Expect(actualError).NotTo(HaveOccurred())
			})

			It("should call BatchCreateOccurrences a single time", func() {
				Expect(rodeClient.BatchCreateOccurrencesCallCount()).To(Equal(1))
			})

			It("should send a single occurrence", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)

				Expect(actualRequest.Occurrences).To(HaveLen(1))
			})

			It("should set the resource uri using the repository and commit id", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
				expectedResourceUri := "git://github.com/rode/collector-build@" + request.CommitId

				Expect(actualRequest.Occurrences[0].Resource.Uri).To(Equal(expectedResourceUri))
			})

			It("should set a static note name", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
				Expect(actualRequest.Occurrences[0].NoteName).To(Equal("projects/rode/notes/build_collector"))
			})

			It("should set the kind as BUILD", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
				Expect(actualRequest.Occurrences[0].Kind).To(Equal(common_go_proto.NoteKind_BUILD))
			})

			It("should set the provenance id", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.Id).To(Equal(request.ProvenanceId))
			})

			It("should set the build creator", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.Creator).To(Equal(request.Creator))
			})

			It("should set the provenance start, end, and create times", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
				buildProvenance := actualRequest.Occurrences[0].GetBuild().Provenance

				Expect(buildProvenance.StartTime).To(Equal(request.BuildStart))
				Expect(buildProvenance.EndTime).To(Equal(request.BuildEnd))
				Expect(buildProvenance.CreateTime.IsValid()).To(Equal(true))
			})

			It("should set the logsUri in the build provenance", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)

				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.LogsUri).To(Equal(request.LogsUri))
			})

			It("should use the Rode project", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)

				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.ProjectId).To(Equal("projects/rode"))
			})

			It("should set the source context", func() {
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
				source := actualRequest.Occurrences[0].GetBuild().Provenance.SourceProvenance.Context.GetGit()

				Expect(source.Url).To(Equal(request.CommitUri))
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
				_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
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
					_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
					actualStartTime := actualRequest.Occurrences[0].GetBuild().Provenance.StartTime
					Expect(actualStartTime.IsValid()).To(BeTrue())
				})
			})

			Describe("build end is not specified", func() {
				BeforeEach(func() {
					request.BuildEnd = nil
				})

				It("should use the current time", func() {
					_, actualRequest, _ := rodeClient.BatchCreateOccurrencesArgsForCall(0)
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
				expectedError      error
				expectedStatusCode codes.Code
			)

			BeforeEach(func() {
				expectedStatusCode = randomGRPCStatusCode()
				expectedError = status.Errorf(expectedStatusCode, fake.Word())
				rodeClient.BatchCreateOccurrencesReturns(nil, expectedError)
			})

			It("should return an error", func() {
				Expect(actualError).To(HaveOccurred())
				Expect(response).To(BeNil())
			})

			It("should return the status that was returned from rode", func() {
				s := getGRPCStatusFromError(actualError)

				Expect(s.Code()).To(Equal(expectedStatusCode))
				Expect(s.Message()).To(Equal(fmt.Sprintf("Error creating occurrences in Rode: %s", expectedError)))
			})
		})

		Describe("BatchCreateOccurrences does not return expected occurrence", func() {
			BeforeEach(func() {
				rodeClient.BatchCreateOccurrencesReturns(&pb.BatchCreateOccurrencesResponse{}, nil)
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
			BeforeEach(func() {
				rodeClient.UpdateOccurrenceReturns(expectedOccurrence, nil)
				rodeClient.ListOccurrencesReturns(listOccurrencesResponse, nil)
			})

			It("should call ListOccurrences once", func() {
				Expect(rodeClient.ListOccurrencesCallCount()).To(Equal(1))
			})

			It("should call UpdateOccurrence once", func() {
				Expect(rodeClient.UpdateOccurrenceCallCount()).To(Equal(1))
			})

			When("there's a single matching occurrence to update", func() {
				It("should filter all occurrences using the existing artifact", func() {
					_, actualListOccurrencesRequest, _ := rodeClient.ListOccurrencesArgsForCall(0)

					expectedFilter := fmt.Sprintf(`build.provenance.builtArtifacts.nestedFilter(id == "%s")`, request.ExistingArtifactId)

					Expect(actualListOccurrencesRequest.Filter).To(Equal(expectedFilter))
				})

				It("should append the new artifact", func() {
					_, actualUpdateOccurrenceRequest, _ := rodeClient.UpdateOccurrenceArgsForCall(0)
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
					_, actualUpdateOccurrenceRequest, _ := rodeClient.UpdateOccurrenceArgsForCall(0)
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
					_, actualUpdateOccurrenceRequest, _ := rodeClient.UpdateOccurrenceArgsForCall(0)
					Expect(actualUpdateOccurrenceRequest.Id).To(Equal(expectedOccurrenceId))
				})
			})
		})

		Describe("updating the occurrence is unsuccessful", func() {
			var (
				expectedError      error
				expectedStatusCode codes.Code
			)

			BeforeEach(func() {
				expectedStatusCode = randomGRPCStatusCode()
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
					expectedError = status.Errorf(expectedStatusCode, fake.Word())
					rodeClient.ListOccurrencesReturns(nil, expectedError)
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})

				It("should return the status that was returned from rode", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(expectedStatusCode))
					Expect(s.Message()).To(ContainSubstring(expectedError.Error()))
				})
			})

			When("no occurrences have matching artifacts", func() {
				BeforeEach(func() {
					listOccurrencesResponse.Occurrences = []*grafeas_go_proto.Occurrence{}
					rodeClient.ListOccurrencesReturns(listOccurrencesResponse, nil)
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
					expectedError = status.Errorf(expectedStatusCode, fake.Word())
					rodeClient.ListOccurrencesReturns(listOccurrencesResponse, nil)
					rodeClient.UpdateOccurrenceReturns(nil, expectedError)
				})

				It("should return an error", func() {
					Expect(actualError).To(HaveOccurred())
				})

				It("should return the status that was returned from rode", func() {
					s := getGRPCStatusFromError(actualError)

					Expect(s.Code()).To(Equal(expectedStatusCode))
					Expect(s.Message()).To(ContainSubstring("Error updating existing artifact in Rode"))
				})
			})
		})
	})
})

func randomGRPCStatusCode() codes.Code {
	c := []codes.Code{
		codes.Internal,
		codes.InvalidArgument,
		codes.PermissionDenied,
		codes.DeadlineExceeded,
	}

	return c[fake.Number(0, len(c)-1)]
}

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
