package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"github.com/zheka156/market_data/internal/integration/telegram/keyboard_builder"
	"github.com/zheka156/market_data/internal/utils"
)

type CoinInfo struct {
	Quantity  decimal.Decimal
	Price     decimal.Decimal
	Amount    decimal.Decimal
	UpdatedAt time.Time
}

func (bc *BotClient) showMyCoinsCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
	coins, err := bc.Rep.GetChatCoinInfo(chatID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to get coins info, please try again later",
		})
		return
	}
	if coins == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "You don't have any coins. Begin with adding new coins by Start button",
		})
		return
	}

	coinData := make(map[string]CoinInfo)
	for _, coin := range coins {
		price, err := bc.Rep.GetLastHourPriceBySymbol(coin.Coin)
		if err != nil {
			bc.Logger.Sugar().Error("Failed to get price from repository for coin", err)
			price = nil
		}
		amount := utils.CalculateAmount(coin.Quantity, price.Last_price)
		coinData[coin.Coin] = CoinInfo{
			Quantity:  coin.Quantity,
			Price:     price.Last_price,
			Amount:    amount,
			UpdatedAt: price.TS,
		}
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        prepareResponseWithCoinsInfo(coinData),
		ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
	})
}

func (bc *BotClient) mainMenuButtonHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Choose option to start with",
		ReplyMarkup: keyboard_builder.WelcomeKeyboard(),
	})
}

func (bc *BotClient) addOrChangeCoinCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	var message = `
	🔹 Add or Change Coin
Enter a coin and quantity (e.g., BTC,1.234).

If the coin does not exist, it will be added.
If the coin already exists, its quantity will be updated.
Note that some coins could be missing in the repository and added soon.`
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        message,
		ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
	})
}

func (bc *BotClient) addNewCoinCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	inputCoin := utils.MapTickersToQuantityFromUserInput(update.Message.Text)
	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
	chatCoins, err := bc.Rep.GetChatCoinInfo(chatID)
	if err != nil {
		bc.Logger.Sugar().Error("Failed to get coins from repository", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to get coins info, please try again later",
		})
		return
	}
	countOfChatCoins := len(chatCoins)
	chatCoinsList := make([]string, 0, countOfChatCoins)
	for _, v := range chatCoins {
		chatCoinsList = append(chatCoinsList, v.Coin)
	}

	if !bc.isExistingCoin(inputCoin, chatCoinsList) && (countOfChatCoins+1) > MaxChosenOptions {

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("You can't add more than %d coins. Current number is %d: %s", MaxChosenOptions, countOfChatCoins, chatCoinsList),
		})
		return
	}
	for c, q := range inputCoin {
		err := bc.Rep.AddTickerWithQuantityToChat(chatID, c, q)
		if err != nil {
			bc.Logger.Sugar().Error("Failed to add coin with quantity to repository", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Failed to add coin. Please try again later",
			})
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        fmt.Sprintf("✅ Coin *%s* with quantity %s was added or updated", c, q),
			ParseMode:   models.ParseModeMarkdownV1,
			ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
		})
	}
}

func (bc *BotClient) RemoveCoinMessageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	var message = `
	🪙Your coins: *%s*
	❌ To remove specific coins: Remove BTC,ETH
	🗑️To Remove all coins, type: Remove all
	`

	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
	var coinsToDelete []string
	coins, err := bc.Rep.GetChatCoinInfo(chatID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to get coins info, please try again later",
		})
		return
	}
	for _, c := range coins {
		coinsToDelete = append(coinsToDelete, c.Coin)
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        fmt.Sprintf(message, coinsToDelete),
		ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
	})
}

func (bc *BotClient) isExistingCoin(inputCoin map[string]decimal.Decimal, coins []string) bool {
	for _, coin := range coins {
		for c := range inputCoin {
			if coin == c {
				return true
			}
		}
	}
	return false
}

func (bc *BotClient) removeCoinCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	removedCoinsMessage := `
	✅ Coins *%s* removed from this chat
	`

	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
	userInput := strings.ToUpper(update.Message.Text)
	if userInput == "REMOVE ALL" {
		coinsToDelete, err := bc.Rep.GetChatCoinInfo(chatID)
		if err != nil {
			bc.Logger.Sugar().Error("Failed to get coins from repository", err)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Failed to get coins info, please try again later",
			})
			return
		}
		for _, c := range coinsToDelete {
			err := bc.Rep.RemoveCoinFromChat(chatID, c.Coin)
			if err != nil {
				bc.Logger.Sugar().Error("Failed to remove coin from repository", err)
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Failed to remove coin. Please try again later",
				})
				return
			}
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "✅ All coins were removed",
			ParseMode:   models.ParseModeMarkdownV1,
			ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
		})
	} else {
		coinsToDelete := strings.TrimPrefix(userInput, "REMOVE")

		coinsList := utils.GetTickersFromUserInput(coinsToDelete)

		found, _ := findCoinsInRepo(coinsList)
		if len(found) == 0 {
			bc.Logger.Info("No coins found for removal")
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "No coins found for removal",
			})
			return
		}

		for _, c := range found {
			err := bc.Rep.RemoveCoinFromChat(chatID, c)
			if err != nil {
				bc.Logger.Sugar().Error("Failed to remove coin from repository", err)
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Failed to remove coin. Please try again later",
				})
				return
			}
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        fmt.Sprintf(removedCoinsMessage, found),
			ParseMode:   models.ParseModeMarkdownV1,
			ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
		})
	}
}
