package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	newLogger "github.com/zheka156/market_data/internal/common/log"
	"github.com/zheka156/market_data/internal/config"
	"github.com/zheka156/market_data/internal/integration/binance"
	"github.com/zheka156/market_data/internal/integration/polygon"
	"github.com/zheka156/market_data/internal/integration/telegram"
	"github.com/zheka156/market_data/internal/job"
	"github.com/zheka156/market_data/internal/middleware"
	"github.com/zheka156/market_data/internal/postgres"
	"github.com/zheka156/market_data/internal/server"
	"go.uber.org/zap"
)

const ConfigPath = "./configs/app.yaml"

func main() {

	config := config.LoadConfig(ConfigPath)

	logger, err := newLogger.NewLogger()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %s", err)
	}
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	webApp := middleware.New(logger)

	polygonClient := polygon.NewClient(logger)
	binanceClient := binance.NewClient(logger, config)
	dbClient := postgres.NewClient(logger)

	server := server.NewServer(polygonClient, binanceClient, dbClient, logger)
	server.InitRoutes(webApp)

	go job.HourJob(*job.NewJobParams(logger, binanceClient, dbClient))

	go telegram.NewBot(ctx, logger, dbClient)

	port := os.Getenv("PORT")
	go func() {
		logger.Sugar().Infof("http server is starting on port %s", port)
		if err := webApp.Listen(":" + port); err != nil {
			logger.Sugar().Fatalf("failed to listen: %s", err.Error())
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	<-signalChan

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := webApp.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Sugar().Errorf("Failed to shutdown server: %v", zap.Error(err))
	} else {
		logger.Info("http server is shut down")
	}

	cancel()
	logger.Info("Bot is shut down")

}
