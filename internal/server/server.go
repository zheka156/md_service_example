package server

import (
	"github.com/zheka156/market_data/internal/integration/binance"
	"github.com/zheka156/market_data/internal/integration/polygon"
	"github.com/zheka156/market_data/internal/postgres"
	"go.uber.org/zap"
)

type Server struct {
	polygonClient *polygon.Client
	binanceClient *binance.Client
	rep postgres.Repository
	log *zap.Logger
}

func NewServer(polygonClient *polygon.Client, 
	binanceClient *binance.Client, db postgres.Repository, logger *zap.Logger) *Server {
	return &Server{
		polygonClient: polygonClient,
		binanceClient: binanceClient,
		rep: db,
		log: logger,
	}
}