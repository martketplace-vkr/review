package app

import (
	"context"
	"time"

	catalogclient "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/client"
	orderclient "github.com/martketplace-vkr/order/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/pkg/build"
	"github.com/martketplace-vkr/pkg/build/components/pgxsqlxcomponent"
	"github.com/martketplace-vkr/review/config"
	serverCmp "github.com/martketplace-vkr/review/internal/app/cmp/server"
	reviewRepository "github.com/martketplace-vkr/review/internal/repository/pg"
	reviewService "github.com/martketplace-vkr/review/internal/service/review"
	adminTransport "github.com/martketplace-vkr/review/internal/transport/grpc/v1/admin"
	clientTransport "github.com/martketplace-vkr/review/internal/transport/grpc/v1/client"
	vendorTransport "github.com/martketplace-vkr/review/internal/transport/grpc/v1/vendor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Run(ctx context.Context, cfg *config.Config) error {
	pg := pgxsqlxcomponent.New(cfg.Postgres)

	catalogConn, err := dialGRPC(ctx, cfg.Catalog)
	if err != nil {
		return err
	}
	defer catalogConn.Close()

	orderConn, err := dialGRPC(ctx, cfg.Order)
	if err != nil {
		return err
	}
	defer orderConn.Close()

	repo := reviewRepository.New(pg.DB)
	service := reviewService.New(
		repo,
		catalogclient.NewCatalogClientServiceClient(catalogConn),
		orderclient.NewOrderClientServiceClient(orderConn),
		cfg.Catalog.Timeout.Duration,
		cfg.Order.Timeout.Duration,
	)

	clientHandler := clientTransport.New(service)
	vendorHandler := vendorTransport.New(service)
	adminHandler := adminTransport.New(service)
	grpcServer := serverCmp.New(cfg.Grpc, clientHandler, vendorHandler, adminHandler)

	cmps := build.Components{
		pg,
		grpcServer,
	}

	app, err := build.NewApp(cmps)
	if err != nil {
		return err
	}

	return build.Run(ctx, app)
}

func dialGRPC(ctx context.Context, cfg config.GRPCClient) (*grpc.ClientConn, error) {
	timeout := cfg.Timeout.Duration
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return grpc.DialContext(
		dialCtx,
		cfg.Host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}
