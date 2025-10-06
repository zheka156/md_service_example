package polygon

import (
	"context"
	"net/http"
	"os"
	"time"

	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/zheka156/market_data/internal/common/log"
	"go.uber.org/zap"
)

type Client struct {
	*polygon.Client
	logger *zap.Logger
}

func NewClient(logger *zap.Logger) *Client {
	client := &http.Client{
		Transport: &log.LoggingRoundTripper{
			Proxied: http.DefaultTransport,
			Logger:  logger,
		},
	}
	c := polygon.NewWithClient(os.Getenv("POLYGON_TKN"), client)
	c.Client.HTTP.Debug = true
	return &Client{c, logger}
}

func (c *Client) GetLastDatePrices(ticker string) *models.GetDailyOpenCloseAggResponse {
	yesterday := time.Now().AddDate(0, 0, -1)
	params := &models.GetDailyOpenCloseAggParams{
		Ticker: ticker,
		Date:   models.Date(yesterday),
	}

	resp, err := c.GetDailyOpenCloseAgg(context.Background(), params, models.WithTrace(true))
	if err != nil {
		c.logger.Error("failed to get last date prices", zap.Error(err))
	}
	return resp
}
