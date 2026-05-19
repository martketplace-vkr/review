package config

import (
	"github.com/martketplace-vkr/pkg/build/components/pgxsqlxcomponent"
	"github.com/martketplace-vkr/pkg/server/grpc"
	"github.com/martketplace-vkr/pkg/utils/duration"
)

type GRPCClient struct {
	Host    string           `validate:"required"`
	Timeout duration.Seconds `validate:"required" default:"5"`
}

type Config struct {
	Grpc     grpc.Config             `validate:"required"`
	Postgres pgxsqlxcomponent.Config `validate:"required"`
	Catalog  GRPCClient              `validate:"required"`
	Order    GRPCClient              `validate:"required"`
}
