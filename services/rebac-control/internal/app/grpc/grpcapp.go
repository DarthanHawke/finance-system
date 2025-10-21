package grpcapp

import (
	grpcaccount "billing-service/internal/grpc/account"
	grpcauth "billing-service/internal/grpc/auth"
	"billing-service/internal/grpc/interceptor"
	grpcrole "billing-service/internal/grpc/role"
	grpctransaction "billing-service/internal/grpc/transaction"
	grpcuser "billing-service/internal/grpc/user"

	"crypto/tls"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type App struct {
	logger     *zap.Logger
	gRPCServer *grpc.Server
	gRPCport   int
}

func New(
	logger *zap.Logger,
	gRPCport int,
	tlsConfig *tls.Config,
	authInterceptor *interceptor.AuthInterceptor,
	authService grpcauth.Auth,
	transactionService grpctransaction.Transaction,
	accountService grpcaccount.Account,
	userService grpcuser.User,
	roleService grpcrole.Role,

) *App {
	creds := credentials.NewTLS(tlsConfig)
	gRPCServer := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor.Unary()), grpc.Creds(creds))
	grpcauth.NewAuthServer(gRPCServer, authService)
	grpctransaction.NewTransactionServer(gRPCServer, transactionService)
	grpcaccount.NewAccountServer(gRPCServer, accountService)
	grpcuser.NewUserServer(gRPCServer, userService)
	grpcrole.NewRoleServer(gRPCServer, roleService)
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
	a.logger.Info("Starting gRPC server", zap.Int("port", a.gRPCport))

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.gRPCport))
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	a.logger.Info("gRPC server is running", zap.String("addres", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (a *App) Stop() {
	a.logger.Info("Stopping gRPC server")
	a.gRPCServer.GracefulStop()
}
