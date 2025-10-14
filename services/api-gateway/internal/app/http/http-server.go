package http

import (
	accounthandler "client-service/internal/http/handlers/account"
	authhandler "client-service/internal/http/handlers/auth"
	paymenthandler "client-service/internal/http/handlers/payment"
	"os"

	rolehandler "client-service/internal/http/handlers/role"
	userhandler "client-service/internal/http/handlers/user"

	mymddl "client-service/internal/http/middleware"
	"context"
	"fmt"
	"net/http"
	"time"

	_ "client-service/docs"

	"github.com/go-chi/chi/v5"

	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	"go.uber.org/zap"
)

type HTTPApp struct {
	logger *zap.Logger
	server *http.Server
}

func New(
	logger *zap.Logger,
	httpPort int,
	authService authhandler.AuthService,
	paymentService paymenthandler.PaymentService,
	accountService accounthandler.AccountService,
	userService userhandler.UserService,
	roleService rolehandler.RoleService,
) *HTTPApp {
	authHandler := authhandler.NewAuthHandler(authService)
	paymentHandler := paymenthandler.NewPaymentHandler(paymentService)
	accountHandler := accounthandler.NewAccountHandler(accountService)
	userHandler := userhandler.NewUserHandler(userService)
	roleHandler := rolehandler.NewRoleHandler(roleService)

	// Роутер
	router := chi.NewRouter()

	router.Use(mymddl.BillingAuthMiddleware(authService))
	router.Use(mymddl.UserInfoMiddleware)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://localhost*", "http://localhost*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API маршруты
	router.Route("/", func(apiRouter chi.Router) {
		// Swagger
		apiRouter.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("doc.json"),
		))

		// Маршруты аутентификации (публичные)
		apiRouter.Mount("/auth", authHandler.Routes())

		// Защищенные маршруты
		apiRouter.Group(func(protectedRouter chi.Router) {
			protectedRouter.Mount("/payments", paymentHandler.Routes())
			protectedRouter.Mount("/account", accountHandler.Routes())
			protectedRouter.Mount("/user", userHandler.Routes())
			protectedRouter.Mount("/role", roleHandler.Routes())
		})
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", httpPort),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &HTTPApp{
		logger: logger,
		server: httpServer,
	}
}

func (a *HTTPApp) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *HTTPApp) Run() error {
	a.logger.Info("Starting HTTP server", zap.String("address", a.server.Addr))

	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}
	return nil
}

func (a *HTTPApp) Stop(ctx context.Context) error {
	a.logger.Info("Stopping HTTP server")
	return a.server.Shutdown(ctx)
}

func serveHTML(file string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if _, err := os.Stat(file); os.IsNotExist(err) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		http.ServeFile(w, r, file)
	}
}
