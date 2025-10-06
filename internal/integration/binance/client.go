package binance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"time"

	"github.com/go-resty/resty/v2"
	"github.com/zheka156/market_data/internal/common/log"
	"github.com/zheka156/market_data/internal/config"
	"go.uber.org/zap"
)

type Binance interface {
	GetLastPrice(ticker string) (string, error)
	GetBatchOfLastPrice(tickers string) ([]Pair, error)
}

type Client struct {
	*resty.Client
	logger *zap.Logger
}

func NewClient(logger *zap.Logger, conf *config.Config) *Client {
	c := resty.New()
	c.SetTransport(&log.LoggingRoundTripper{
		Proxied: http.DefaultTransport,
		Logger:  logger,
	})
	c.SetCloseConnection(true)
	c.SetTimeout(30 * time.Second)
	c.SetHeader("Content-Type", "application/json")
	c.SetBaseURL(os.Getenv("BINANCE_URL"))

	c.OnAfterResponse(func(client *resty.Client, response *resty.Response) error {
		if response.StatusCode() == http.StatusTooManyRequests {
			err := getSleepTimeAndWait(logger, response)
			logger.Sugar().Info("Retrying request")
			if err != nil {
				return err
			}
			client.R().
				SetContext(response.Request.Context()).
				SetBody(response.Request.Body).
				Execute(response.Request.Method, response.Request.URL)
		}
		return nil
	})

	return &Client{c, logger}
}

type GetLastBatchPriceResponse []Pair

type Pair struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

func (c *Client) GetBatchOfLastPrice(tickers string) ([]Pair, error) {

	var response []Pair
	resp, err := c.R().
		SetQueryParam("symbols", tickers).
		Get("/api/v3/ticker/price")
	if err != nil {
		c.logger.Error("Failed to get last price", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		c.logger.Error("Failed to unmarshal response", zap.Error(err))
		return nil, err
	}
	return response, nil
}

func (c *Client) GetLastPrice(ticker string) (string, error) {
	var response Pair
	param := fmt.Sprintf("%sUSDT", ticker)
	resp, err := c.R().
		SetQueryParam("symbol", param).
		Get("/api/v3/ticker/price")
	if err != nil {
		c.logger.Error("Failed to get last price", zap.Error(err))
		return "", err
	}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		c.logger.Error("Failed to unmarshal response", zap.Error(err))
		return "", err
	}
	return response.Price, nil
}

func getSleepTimeAndWait(logger *zap.Logger, response *resty.Response) error {
	sleepTimeString := response.Header().Get("Retry-After")
	logger.Sugar().Warnf("Received signal from binance to sleep for %s seconds", sleepTimeString)

	sleepTime, err := strconv.Atoi(sleepTimeString)
	if err != nil {
		return err
	}
	logger.Sugar().Warnf("Too many requests made to binance, sleep for %d", sleepTime)
	time.Sleep(time.Duration(sleepTime) * time.Second)
	return nil
}
