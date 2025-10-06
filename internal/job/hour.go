package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zheka156/market_data/internal/integration/binance"
	"github.com/zheka156/market_data/internal/postgres"
	"github.com/zheka156/market_data/internal/utils"
	"go.uber.org/zap"
)

const (
	delay              = 1 * time.Hour
	USDT               = "USDT"
	maxTickersPerBatch = 20
)

type JobParams struct {
	Log    *zap.Logger
	Client binance.Binance
	Rep    postgres.Repository
}

func NewJobParams(logger *zap.Logger, client binance.Binance, repository postgres.Repository) *JobParams {
	return &JobParams{
		Log:    logger,
		Client: client,
		Rep:    repository,
	}
}

func HourJob(params JobParams) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := params.Log

	for {
		now := time.Now()
		nextHour := now.Truncate(time.Hour).Add(delay)
		timeUntilNextHour := time.Until(nextHour)

		timer := time.NewTicker(timeUntilNextHour)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			logger.Info("Hourly job started")
			err := params.Process()
			if err != nil {
				logger.Error("Failed to update hourly price", zap.Error(err))
				continue
			}
			logger.Info("Hourly job stopped")
		}
	}
}

func (p JobParams) Process() error {

	coins, err := p.Rep.GetTickers()
	if err != nil {
		return err
	}

	//retrieve prices for coins in batches, store the whole batch in memory
	var queryParam string
	var batchOfPrices []binance.Pair
	for i := 0; i < len(coins); i += maxTickersPerBatch {

		end := i + maxTickersPerBatch
		if end > len(coins) {
			end = len(coins)
		}
		chunk := coins[i:end]

		queryParam = prepareQueryParamForBatch(chunk)

		response, err := p.Client.GetBatchOfLastPrice(queryParam)
		if err != nil {
			p.Log.Sugar().Errorf("Failed to retrieve price for batch %v", chunk, zap.Error(err))
			return err
		}
		batchOfPrices = append(batchOfPrices, response...)
	}
	p.Log.Debug("Batch of prices retrieved", zap.Any("count", batchOfPrices))

	//to do: make batch insert
	for _, pair := range batchOfPrices {
		price, err := utils.StringToDecimal(pair.Price)
		if err != nil {
			p.Log.Error("Failed to convert price to decimal", zap.Error(err))
			return err
		}
		p.Log.Debug("Inserting hourly price", zap.String("symbol", pair.Symbol), zap.String("price", price.String()))
		err = p.Rep.InsertPrice(&postgres.Price{
			Fromsymbol: strings.TrimSuffix(pair.Symbol, "USDT"),
			Last_price: price.Truncate(8),
			TS:         time.Now(),
			Tosymbol:   USDT,
		})
		if err != nil {
			p.Log.Error("Failed to insert hourly price", zap.Error(err))
			return err
		}
	}
	return nil
}

func prepareQueryParamForBatch(chunk []string) string {
	var coinsToRequest []string
	for _, ticker := range chunk {
		coinsToRequest = append(coinsToRequest, fmt.Sprintf("\"%sUSDT\"", ticker))
	}
	queryParam := strings.Join(coinsToRequest, ",")
	queryParam = fmt.Sprintf("[%s]", queryParam)
	return queryParam
}
