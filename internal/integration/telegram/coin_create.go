package telegram

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/zheka156/market_data/internal/integration/telegram/keyboard_builder"
	"github.com/zheka156/market_data/internal/utils"
)

var popularOptions = []string{"BTC", "ETH", "DOGE"}
var chosenCoinOptions = []string{}
var uniqueUserCoinsToSave = []string{}

const (
	MaxChosenOptions      = 10
	InputQuantityTemplate = `
Enter the quantity for your selected coins: %s.
✅ Use . as the decimal separator (e.g., 1.23456789).
✅ Separate multiple coins with , (e.g., 0.5,1.2).
🔢 Supports up to 8 decimal places.
`

	manualInputTemplate = `
	Your coins list: *%s*
Input up to 10 coins in total to check their presence. Separate them with a comma.
Example: BTC,eth,ada.
`
	manualInputCoinsResultTemplate = `
✅ Found coins: *%s*
⚠️ Not found: *%s*
✏️ To modify the list, enter new coins.
`
)

func (bc *BotClient) chooseCoinsCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	switch update.CallbackQuery.Data {
	case "btn_opt1":
		chosenCoinOptions = addRemoveSelection(popularOptions[0], chosenCoinOptions)
	case "btn_opt2":
		chosenCoinOptions = addRemoveSelection(popularOptions[1], chosenCoinOptions)
	case "btn_opt3":
		chosenCoinOptions = addRemoveSelection(popularOptions[2], chosenCoinOptions)
	case "btn_manual":
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			Text:      fmt.Sprintf(manualInputTemplate, chosenCoinOptions),
			ParseMode: models.ParseModeMarkdownV1,
		})
		return
	case "btn_select_coins":
		b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
		})
		if len(chosenCoinOptions) == 0 {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
				Text:        "No coins selected. Please select at least one coin",
				ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
			})
			return
		}
		availableCoins := bc.trimCoinsAccordingRepository(chosenCoinOptions, strconv.FormatInt(update.CallbackQuery.Message.Message.Chat.ID, 10))
		uniqueUserCoinsToSave = availableCoins
		message := fmt.Sprintf(InputQuantityTemplate, availableCoins)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
			Text:        message,
			ReplyMarkup: keyboard_builder.EraseCoinKeyboard(),
		})
		return
	}
}

func addRemoveSelection(text string, chosenOptions []string) []string {
	if len(chosenOptions) == 0 {
		return append(chosenOptions, text)
	}
	for i := range chosenOptions {
		if text == chosenOptions[i] {
			return append(chosenOptions[:i], chosenOptions[i+1:]...)
		}
	}
	return append(chosenOptions, text)
}

func (bc *BotClient) selectCoinsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        fmt.Sprintf("Select coins from the list or input manually - up to %d in total", MaxChosenOptions),
		ReplyMarkup: keyboard_builder.CoinsKeyboard(popularOptions),
	})
}

func (bc *BotClient) manualCoinInputMessageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	userCoins := utils.GetTickersFromUserInput(update.Message.Text)
	mergedCoins := append(userCoins, chosenCoinOptions...)
	uniqueCoins := utils.RemoveDuplicates(mergedCoins)
	found, notFound := findCoinsInRepo(uniqueCoins)

	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)

	if len(notFound) != 0 {
		bc.Logger.Sugar().Warnf("Coins %s were not found in repository", notFound)
	}

	uniqueUserCoinsToSave = bc.trimCoinsAccordingRepository(found, chatID)
	if len(uniqueUserCoinsToSave) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        fmt.Sprintf("You can't add more than %d coins", MaxChosenOptions),
			ReplyMarkup: keyboard_builder.ManageCoinsKeyboard(),
		})
		return
	}
	messageToInputQuantity := fmt.Sprintf(InputQuantityTemplate, uniqueUserCoinsToSave)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        fmt.Sprintf(manualInputCoinsResultTemplate, uniqueUserCoinsToSave, notFound) + messageToInputQuantity,
		ParseMode:   models.ParseModeMarkdownV1,
		ReplyMarkup: keyboard_builder.EraseCoinKeyboard(),
	})
}

func findCoinsInRepo(input []string) (found, notFound []string) {
	for _, val := range input {
		if _, ok := repositoryCoins[val]; ok {
			found = append(found, val)
			continue
		} else {
			notFound = append(notFound, val)
		}
	}
	return found, notFound
}

func (bc *BotClient) eraseInputCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	chosenCoinOptions = []string{}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Select coins from the list or choose option to input manually",
		ReplyMarkup: keyboard_builder.CoinsKeyboard(popularOptions),
	})
}

func (bc *BotClient) trimCoinsAccordingRepository(input []string, chatID string) []string {
	userRepCoins, err := bc.Rep.GetChatCoinInfo(chatID)
	if err != nil {
		bc.Logger.Sugar().Error("Failed to get coins from repository", err)
		return nil
	}
	if len(userRepCoins)+len(input) <= MaxChosenOptions {
		return input
	}

	availableCount := MaxChosenOptions - len(userRepCoins)
	return input[:availableCount]
}
