package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	"aziz.dev/redirect/internal/pb"
	"aziz.dev/redirect/internal/resolve"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	pb.UnimplementedRedirectServiceServer
	service resolve.Service
}

func NewGRPCServer(service resolve.Service) *grpcServer {
	return &grpcServer{service: service}
}

func StartGRPC(service resolve.Service, port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", port, err)
	}

	srv := grpc.NewServer()
	pb.RegisterRedirectServiceServer(srv, NewGRPCServer(service))

	logrus.WithField("port", port).Info("gRPC server listening")
	return srv.Serve(lis)
}

func (s *grpcServer) ResolveCode(ctx context.Context, req *pb.ResolveCodeRequest) (*pb.ResolveCodeResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	result, err := s.service.ResolveCode(ctx, code)
	if err != nil {
		if errors.Is(err, resolve.ErrLinkInactive) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		if errors.Is(err, resolve.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ResolveCodeResponse{
		Code:        result.Code,
		OriginalUrl: result.OriginalURL,
	}, nil
}
