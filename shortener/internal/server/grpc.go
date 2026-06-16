package server

import (
	"context"
	"fmt"
	"net"

	"aziz.dev/shortener/internal/link"
	"aziz.dev/shortener/internal/pb"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type grpcServer struct {
	pb.UnimplementedShortenerServiceServer
	service link.Service
}

func NewGRPCServer(service link.Service) *grpcServer {
	return &grpcServer{service: service}
}

func StartGRPC(service link.Service, port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", port, err)
	}

	srv := grpc.NewServer()
	pb.RegisterShortenerServiceServer(srv, NewGRPCServer(service))

	logrus.WithField("port", port).Info("gRPC server listening")
	return srv.Serve(lis)
}

func (s *grpcServer) GetAllLinks(ctx context.Context, req *pb.GetAllLinksRequest) (*pb.GetAllLinksResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	links, err := s.service.GetAll(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	resp := &pb.GetAllLinksResponse{Links: make([]*pb.Link, 0, len(links))}
	for i := range links {
		resp.Links = append(resp.Links, dtoToProto(&links[i]))
	}

	return resp, nil
}

func (s *grpcServer) CreateLink(ctx context.Context, req *pb.CreateLinkRequest) (*pb.CreateLinkResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	createReq := link.CreateRequest{
		OriginalURL: req.GetOriginalUrl(),
	}
	if req.CustomAlias != nil {
		createReq.CustomAlias = *req.CustomAlias
	}
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.AsTime()
		createReq.ExpiresAt = &t
	}

	created, err := s.service.Create(ctx, userID, createReq)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.CreateLinkResponse{Link: dtoToProto(created)}, nil
}

func (s *grpcServer) UpdateExpiry(ctx context.Context, req *pb.UpdateExpiryRequest) (*pb.UpdateExpiryResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if req.ExpiresAt == nil {
		return nil, status.Error(codes.InvalidArgument, "expires_at is required")
	}

	expiresAt := req.ExpiresAt.AsTime()
	updateReq := link.UpdateExpiryDto{ExpiresAt: &expiresAt}

	if err := s.service.UpdateExpiry(ctx, userID, req.GetCode(), updateReq); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.UpdateExpiryResponse{Message: "successfully updated expiry"}, nil
}

func (s *grpcServer) DeleteLink(ctx context.Context, req *pb.DeleteLinkRequest) (*pb.DeleteLinkResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.service.Delete(ctx, userID, req.GetCode()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.DeleteLinkResponse{Message: "successfully deleted link"}, nil
}

func dtoToProto(d *link.Dto) *pb.Link {
	l := &pb.Link{
		Code:        d.Code,
		OriginalUrl: d.OriginalURL,
		IsActive:    d.IsActive,
		CreatedAt:   timestamppb.New(d.CreatedAt),
	}
	if d.CustomAlias != nil {
		l.CustomAlias = d.CustomAlias
	}
	if d.ExpiresAt != nil {
		l.ExpiresAt = timestamppb.New(*d.ExpiresAt)
	}
	return l
}
