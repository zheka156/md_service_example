package keyboard_builder

import "github.com/go-telegram/bot/models"

func WelcomeKeyboard() models.ReplyKeyboardMarkup {

	kb := models.ReplyKeyboardMarkup{
		OneTimeKeyboard: true,
		ResizeKeyboard:  true,
		IsPersistent:    true,
		Keyboard: [][]models.KeyboardButton{
			{
				{Text: "Start with new coins"},
				{Text: "Show my coins"},
			},
		},
	}
	return kb
}

func ManageCoinsKeyboard() models.ReplyKeyboardMarkup {
	kb := models.ReplyKeyboardMarkup{
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
		Keyboard: [][]models.KeyboardButton{
			{
				{Text: "Add/Change coin"},
				{Text: "Remove coin"},
			},
			{
				{Text: "Get Coin updates"},
				{Text: "Home"},
			},
		},
	}
	return kb
}

func CoinsKeyboard(popularOptions []string) models.ReplyMarkup {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: popularOptions[0], CallbackData: "btn_opt1"},
				{Text: popularOptions[1], CallbackData: "btn_opt2"},
				{Text: popularOptions[2], CallbackData: "btn_opt3"},
			}, {
				{Text: "Select", CallbackData: "btn_select_coins"},
				{Text: "Input Manually", CallbackData: "btn_manual"},
			},
		},
	}
	return kb
}

func EraseCoinKeyboard() models.ReplyKeyboardMarkup {
	kb := models.ReplyKeyboardMarkup{
		OneTimeKeyboard:       true,
		ResizeKeyboard:        true,
		InputFieldPlaceholder: "Enter coin Or quantity",
		Keyboard: [][]models.KeyboardButton{
			{
				{Text: "Erase input"},
			}}}
	return kb
}
