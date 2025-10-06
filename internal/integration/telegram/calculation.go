package telegram

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"github.com/zheka156/market_data/internal/integration/telegram/keyboard_builder"
	"github.com/zheka156/market_data/internal/utils"
)

const coinInfoTemplate = `
🔹%s🔹%s tokens worth %s USDT
`

func (bc *BotClient) provideCalculationByQuantityCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	userInput := update.Message.Text
	if len(uniqueUserCoinsToSave) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "You didn't input coins to calculate. Start with new coins",
			ReplyMarkup: keyboard_builder.WelcomeKeyboard(),
		})
		return
	}

	quantityList := strings.Split(userInput, ",")
	if len(quantityList) != len(uniqueUserCoinsToSave) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Number of quantities should be equal to number of coins. Review the list and input again",
		})
		return
	}

	coinStorage := make(map[string]CoinInfo)
	for i, val := range quantityList {
		quantity, err := decimal.NewFromString(val)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Wrong format for quantity" + val,
			})
			return
		}

		price, err := bc.Rep.GetLastHourPriceBySymbol(uniqueUserCoinsToSave[i])
		if err != nil {
			bc.Logger.Sugar().Error("Failed to get price from repository for coin", err)
			price = nil
		}
		amount := utils.CalculateAmount(quantity, price.Last_price)

		coinStorage[uniqueUserCoinsToSave[i]] = CoinInfo{
			Quantity:  quantity,
			Price:     price.Last_price,
			Amount:    amount,
			UpdatedAt: price.TS,
		}
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   prepareResponseWithCoinsInfo(coinStorage),
		ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
	})

	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
	err := bc.Rep.CreateChat(chatID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to save chat, please try again later",
		})
	}
	for k, v := range coinStorage {
		err := bc.Rep.CreateChatCoins(chatID, k, v.Quantity)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Failed to save your coins, please try again later",
			})
		}
	}
}

func prepareResponseWithCoinsInfo(coins map[string]CoinInfo) string {
	var response string
	var updatedAt time.Time
	var sumOfAmounts decimal.Decimal

	keys := make([]string, 0, len(coins))
    for k := range coins {
        keys = append(keys, k)
    }
	sort.Strings(keys)

	for _, k := range keys {
		v := coins[k]
		response += fmt.Sprintf(coinInfoTemplate, k, v.Quantity.Round(2).String(), v.Amount.String())
		sumOfAmounts = sumOfAmounts.Add(v.Amount)
		updatedAt = v.UpdatedAt
	}
	response += fmt.Sprintf(`
	📊Total Value: %s USDT
	📅 Updated at: %s UTC`, sumOfAmounts.String(), updatedAt.Format("2006-01-02 15:04"))
	return response
}
