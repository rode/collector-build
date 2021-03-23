package server

import (
	"context"

	"github.com/rode/collector-build/proto/v1alpha1"
	pb "github.com/rode/rode/proto/v1alpha1"
	"go.uber.org/zap"
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
	return &v1alpha1.CreateBuildResponse{}, nil
}
