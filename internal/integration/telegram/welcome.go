package telegram

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/zheka156/market_data/internal/integration/telegram/keyboard_builder"
)

const welcomeMessage = `🚀 Welcome to CryptoTracker! 🚀
Keep a track of your crypto investments! Just select your cryptocurrency, enter the quantity, and let me calculate its latest value for you. 📊💰

🔹 Stay updated with latest prices
🔹 Track your portfolio effortlessly

Ready to get started? Choose your first cryptocurrency now! 🚀📈`

func (bc *BotClient) welcomeHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	// reset the chosen options in case the chat is restarted
	chosenCoinOptions = []string{}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        welcomeMessage,
		ReplyMarkup: keyboard_builder.WelcomeKeyboard(),
	})
}
