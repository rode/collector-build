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
	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/collector-build/mocks"
	"github.com/rode/collector-build/proto/v1alpha1"
	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
				Artifacts: []string{
					fake.URL(),
					fake.URL(),
				},
				CommitId:   fake.LetterN(10),
				Repository: fake.URL(),
			}
		})

		JustBeforeEach(func() {
			response, actualError = server.CreateBuild(ctx, request)
		})

		Describe("successful", func() {
			var (
				expectedOccurrenceId string
				expectedProvenanceId string
				actualRequest        *pb.BatchCreateOccurrencesRequest
			)

			BeforeEach(func() {
				expectedOccurrenceId = fake.LetterN(10)
				newOccurrence := &grafeas_go_proto.Occurrence{
					Name: expectedOccurrenceId,
				}

				expectedProvenanceId = fake.UUID()
				newUuid = func() uuid.UUID {
					return uuid.MustParse(expectedProvenanceId)
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

			It("should generate a new id for the provenance", func() {
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.Id).To(Equal(expectedProvenanceId))
			})

			It("should use the Rode project", func() {
				Expect(actualRequest.Occurrences[0].GetBuild().Provenance.ProjectId).To(Equal("projects/rode"))
			})

			It("should set the source context", func() {
				source := actualRequest.Occurrences[0].GetBuild().Provenance.SourceProvenance.Context.GetGit()

				Expect(source.Url).To(Equal(request.Repository))
				Expect(source.RevisionId).To(Equal(request.CommitId))
			})
		})

		Describe("sad", func() {

			Context("request validation", func() {
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
						Expect(s.Message()).To(ContainSubstring("Invalid Repository URL"))
					})
				})

				When("the request contains no artifacts", func() {
					BeforeEach(func() {
						request.Artifacts = []string{}
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
	})
})

func getGRPCStatusFromError(err error) *status.Status {
	s, ok := status.FromError(err)
	Expect(ok).To(BeTrue(), "Expected error to be a gRPC status")

	return s
}
