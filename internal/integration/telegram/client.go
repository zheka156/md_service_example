package telegram

import (
	"context"
	"os"
	"regexp"

	"github.com/go-telegram/bot"
	"github.com/zheka156/market_data/internal/postgres"
	"github.com/zheka156/market_data/internal/utils"
	"go.uber.org/zap"
)

type BotClient struct {
	*bot.Bot
	Logger *zap.Logger
	Rep    postgres.Repository
}

var repositoryCoins = make(map[string]struct{})

func NewBot(ctx context.Context, logger *zap.Logger, rep postgres.Repository) {

	token := os.Getenv("TG_TKN")

	// opts := bot.WithMessageTextHandler("/start", bot.MatchTypeExact, welcomeHandler)

	b, err := bot.New(token)
	if err != nil {
		logger.Sugar().Errorf("failed to create telegram bot", err)
	}
	bc := &BotClient{
		b,
		logger,
		rep,
	}

	coinsList, err := rep.GetTickers()
	if err != nil {
		logger.Sugar().Errorf("failed to get tickers from repository", err)
	}

	//creating cache of coins
	for _, c := range coinsList {
		repositoryCoins[c] = struct{}{}
	}
	logger.Info("Loaded cache of coins")

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, bc.welcomeHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/select", bot.MatchTypeExact, bc.selectCoinsHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "Start with new coins", bot.MatchTypeExact, bc.selectCoinsHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "btn_", bot.MatchTypePrefix, bc.chooseCoinsCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "Show my coins", bot.MatchTypeExact, bc.showMyCoinsCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "Get Coin updates", bot.MatchTypeExact, bc.showMyCoinsCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "Add/Change coin", bot.MatchTypeExact, bc.addOrChangeCoinCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "Erase input", bot.MatchTypeExact, bc.eraseInputCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "Home", bot.MatchTypeExact, bc.mainMenuButtonHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "Input Manually", bot.MatchTypeExact, bc.manualCoinInputMessageHandler)

	numberRegexp := regexp.MustCompile(`^\d+(\.\d+)?(,\d+(\.\d+)?)*$`)
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, numberRegexp, bc.provideCalculationByQuantityCommandHandler)


	b.RegisterHandler(bot.HandlerTypeMessageText, "Remove coin", bot.MatchTypeExact, bc.RemoveCoinMessageHandler)
	removalRegexp, _ := utils.CreateRemoveRegexp()
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, removalRegexp, bc.removeCoinCommandHandler)

	coinQuantityRegexp, _ := utils.CreateCoinQuantityRegexp(coinsList)
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, coinQuantityRegexp, bc.addNewCoinCommandHandler)

	coinRegexp, _ := utils.CreateCoinRegexp(coinsList)
	b.RegisterHandlerRegexp(bot.HandlerTypeMessageText, coinRegexp, bc.manualCoinInputMessageHandler)

	bc.Logger.Info("Starting telegram bot")
	bc.Start(ctx)
}
