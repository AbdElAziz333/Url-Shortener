package grpcclient

import (
	"fmt"

	redirectpb "aziz.dev/gateway/internal/pb/redirect"
	shortenerpb "aziz.dev/gateway/internal/pb/shortener"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	Shortener shortenerpb.ShortenerServiceClient
	Redirect  redirectpb.RedirectServiceClient
	conns     []*grpc.ClientConn
}

func NewClients(shortenerAddr, redirectAddr string) (*Clients, error) {
	shortenerConn, err := grpc.NewClient(
		shortenerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(circuitBreakerUnaryInterceptor("shortener")),
	)
	if err != nil {
		return nil, fmt.Errorf("dial shortener: %w", err)
	}

	redirectConn, err := grpc.NewClient(
		redirectAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(circuitBreakerUnaryInterceptor("redirect")),
	)
	if err != nil {
		_ = shortenerConn.Close()
		return nil, fmt.Errorf("dial redirect: %w", err)
	}

	return &Clients{
		Shortener: shortenerpb.NewShortenerServiceClient(shortenerConn),
		Redirect:  redirectpb.NewRedirectServiceClient(redirectConn),
		conns:     []*grpc.ClientConn{shortenerConn, redirectConn},
	}, nil
}

func (c *Clients) Close() error {
	var firstErr error
	for _, conn := range c.conns {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
