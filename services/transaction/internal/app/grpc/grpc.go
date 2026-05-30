package grpcapp

import (
	"log/slog"
	grpcinterceptor "transaction-service/internal/transport/grpc/interceptor"
	grpctransaction "transaction-service/internal/transport/grpc/server"

	"fmt"
	"net"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	logger     *slog.Logger
	gRPCServer *grpc.Server
	gRPCport   int
}

func New(
	logger *slog.Logger,
	gRPCport int,
	transactionService grpctransaction.Transaction,
) *App {
	gRPCServer := grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(grpcinterceptor.UnaryServerInterceptor(logger)),
	)
	grpctransaction.NewTransactionServer(gRPCServer, transactionService)
	return &App{
		logger:     logger,
		gRPCServer: gRPCServer,
		gRPCport:   gRPCport,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	a.logger.Info("Starting gRPC server", "port", a.gRPCport)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.gRPCport))
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	a.logger.Info("gRPC server is running", "addres", l.Addr().String())

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (a *App) Stop() {
	a.logger.Info("Stopping gRPC server")
	a.gRPCServer.GracefulStop()
}
