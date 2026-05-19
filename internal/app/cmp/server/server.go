package server

import (
	"context"
	"net"
	"time"

	grpcServer "github.com/martketplace-vkr/pkg/server/grpc"
	adminpb "github.com/martketplace-vkr/review/pkg/api/grpc/v1/admin"
	clientpb "github.com/martketplace-vkr/review/pkg/api/grpc/v1/client"
	vendorpb "github.com/martketplace-vkr/review/pkg/api/grpc/v1/vendor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const cmpName = "GRPC server"

type Server struct {
	cfg        grpcServer.Config
	grpcServer *grpc.Server
	client     clientpb.ReviewClientServiceServer
	vendor     vendorpb.ReviewVendorServiceServer
	admin      adminpb.ReviewAdminServiceServer
}

func New(
	cfg grpcServer.Config,
	client clientpb.ReviewClientServiceServer,
	vendor vendorpb.ReviewVendorServiceServer,
	admin adminpb.ReviewAdminServiceServer,
) *Server {
	return &Server{
		cfg:    cfg,
		client: client,
		vendor: vendor,
		admin:  admin,
	}
}

func (s *Server) Start(ctx context.Context) (err error) {
	server, err := grpcServer.New(ctx, s.cfg, nil)
	if err != nil {
		return err
	}

	s.grpcServer = server.Grpc
	reflection.Register(s.grpcServer)
	clientpb.RegisterReviewClientServiceServer(s.grpcServer, s.client)
	vendorpb.RegisterReviewVendorServiceServer(s.grpcServer, s.vendor)
	adminpb.RegisterReviewAdminServiceServer(s.grpcServer, s.admin)

	listener, err := net.Listen("tcp", s.cfg.Host)
	if err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(s.cfg.StartTimeout.Duration):
		return nil
	}
}

func (s *Server) Stop(_ context.Context) error {
	stopCh := make(chan any)
	go func() {
		s.grpcServer.GracefulStop()
		stopCh <- nil
	}()

	select {
	case <-time.After(s.cfg.StopTimeout.Duration):
		return nil
	case <-stopCh:
		return nil
	}
}

func (s *Server) GetName() string {
	return cmpName
}

func (s *Server) GetShutdownDelay() time.Duration {
	return time.Second
}

func (s *Server) GetStartTimeout() time.Duration {
	return 5 * time.Second
}

func (s *Server) GetStopTimeout() time.Duration {
	return 5 * time.Second
}
