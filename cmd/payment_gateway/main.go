package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marioarizaj/payment-gateway/internal/config"
	"github.com/marioarizaj/payment-gateway/internal/dependencies"
	"github.com/marioarizaj/payment-gateway/internal/handlers"
	"go.uber.org/zap"
)

func main() {
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		zapLogger.Fatal("could not load config", zap.Error(err))
	}

	deps, err := dependencies.InitDependencies(cfg)
	if err != nil {
		zapLogger.Fatal("could not initialise dependencies", zap.Error(err))
	}

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      handlers.NewRouter(cfg, deps, zapLogger),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	shutdownCompleteChan := handleShutdownSignal(func() {
		// When we call this, ListenAndServe will immediately return
		// http.ErrServerClosed
		if err := server.Shutdown(context.Background()); err != nil {
			zapLogger.Fatal("could not shutdown server", zap.Error(err))
		}
	})
	zapLogger.Info("Server starting to listen on port: ", zap.Int("port", cfg.Server.Port))
	if err = server.ListenAndServe(); err == http.ErrServerClosed {
		// Shutdown has been called, we must wait here until it completes
		<-shutdownCompleteChan
	} else {
		zapLogger.Error("http.ListenAndServer failed", zap.Error(err))
	}

	fmt.Println("INFO: Shutdown gracefully")
}

func handleShutdownSignal(fn func()) <-chan struct{} {
	shutdownSignal := make(chan struct{}, 1)
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fn()

		shutdownSignal <- struct{}{}
	}()
	return shutdownSignal
}
