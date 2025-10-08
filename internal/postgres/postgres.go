package postgres

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Repository interface {
	InsertPrice(price *Price) error
	GetLastHourPriceBySymbol(symbol string) (price *Price, err error)
	GetTickers() ([]string, error)
	CreateChat(chatID string) error
	CreateChatCoins(chatID string, coin string, quantity decimal.Decimal) error
	GetChatCoinInfo(chatID string) ([]*CoinInfo, error)
	GetTicker(inputTicker string) (foundTicker string, err error)
	AddTickerWithQuantityToChat(chatID string, coin string, quantity decimal.Decimal) error
	RemoveCoinFromChat(chatID string, coin string) error
}

type client struct {
	*sqlx.DB
	log *zap.Logger
}

func NewClient(log *zap.Logger) Repository {
	db, err := sqlx.Connect(os.Getenv("DB_ENV"), os.Getenv("DB_URL"))
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		return nil
	}
	return &client{
		db,
		log,
	}
}

type (
	ClientStorage struct {
		ClientID  string `db:"client_id"`
		CreatedAt string `db:"created_at"`
		UpdatedAt string `db:"updated_at"`
	}

	DatabaseStorage struct {
		ID        uuid.UUID `db:"id" name:"id"`
		Status    string    `db:"status" name:"status"`
		ClientID  uuid.UUID `db:"client_id" name:"client_id"`
		CreatedAt time.Time `db:"created_at" name:"created_at"`
		UpdatedAt time.Time `db:"updated_at" name:"updated_at"`
	}

	Page struct {
		ID                 uuid.UUID       `db:"id" name:"id"`
		DatabaseID         string          `db:"database_id" name:"database_id"`
		Ticker             string          `db:"ticker" name:"ticker"`
		Name               string          `db:"name" name:"name"`
		AssetType          string          `db:"asset_type" name:"asset_type"`
		LastPrice          decimal.Decimal `db:"last_price" name:"last_price"`
		ReturnValueUSD     decimal.Decimal `db:"return_value_usd" name:"return_value_usd"`
		LastPriceTimestamp time.Time       `db:"last_price_timestamp" name:"last_price_timestamp"`
		CreatedAt          time.Time       `db:"created_at" name:"created_at"`
		UpdatedAt          time.Time       `db:"updated_at" name:"updated_at"`
	}

	Price struct {
		Fromsymbol string          `db:"fromsym" name:"fromsym"`
		Tosymbol   string          `db:"tosym" name:"tosym"`
		Last_price decimal.Decimal `db:"last_price" name:"last_price"`
		TS         time.Time       `db:"ts" name:"ts"`
	}

	CoinInfo struct {
		Quantity decimal.Decimal `db:"quantity" name:"quantity"`
		Coin     string          `db:"coin" name:"coin"`
		UpdateAt time.Time       `db:"updated_at" name:"updated_at"`
	}
)

func (c *client) InsertPrice(price *Price) error {
	query := `
		INSERT INTO one_hour_price (fromsym, tosym, last_price, ts)
		VALUES (:fromsym, :tosym, :last_price, :ts)
		ON CONFLICT DO NOTHING;
	`
	return c.SafeTx(func(tx *sqlx.Tx) error {
		_, err := tx.NamedExec(query, price)
		if err != nil {
			return fmt.Errorf("failed to insert price for %s->%s: %w", price.Fromsymbol, price.Tosymbol, err)
		}
		return nil
	})
}

func (c *client) GetLastHourPriceBySymbol(symbol string) (price *Price, err error) {
	var prices []*Price
	query := `SELECT fromsym, tosym, last_price, ts FROM one_hour_price WHERE fromsym = $1 ORDER BY ts DESC LIMIT 1`
	err = c.Select(&prices, query, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get last hour price for symbol %s: %w", symbol, err)
	}
	if len(prices) == 0 {
		return nil, fmt.Errorf("no price found for symbol %s", symbol)
	}
	return prices[0], nil
}

func (c *client) GetTickers() ([]string, error) {
	var tickers []string
	query := `SELECT DISTINCT ticker FROM coin ORDER BY ticker`
	err := c.Select(&tickers, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tickers: %w", err)
	}
	return tickers, nil
}

// to delete
func (c *client) GetLastHourUpdateBySymbol(symbol string) (time.Time, error) {
	var lastUpdateTime time.Time
	query := `SELECT ts FROM one_hour_price WHERE fromsym = $1 ORDER BY ts DESC LIMIT 1`
	err := c.Get(&lastUpdateTime, query, symbol)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last hour update for symbol %s: %w", symbol, err)
	}
	return lastUpdateTime, nil
}

func (c *client) CreateChat(chatID string) error {
	query := `
		INSERT INTO tg_chat (chatId, created_at, updated_at, prime, is_active)
		VALUES ($1, NOW(), NOW(), FALSE, TRUE)
		ON CONFLICT (chatId) DO UPDATE SET 
			updated_at = NOW();
	`
	return c.SafeTx(func(tx *sqlx.Tx) error {
		_, err := tx.Exec(query, chatID)
		if err != nil {
			return fmt.Errorf("failed to create chat %s: %w", chatID, err)
		}
		return nil
	})
}

func (c *client) CreateChatCoins(chatID string, coin string, quantity decimal.Decimal) error {
	query := `
			INSERT INTO chat_coins (id, chat_id, coin, quantity) 
			VALUES (gen_random_uuid(), $1, $2, $3)
			ON CONFLICT (chat_id, coin) DO UPDATE SET
			quantity = EXCLUDED.quantity;
		`
	return c.SafeTx(func(tx *sqlx.Tx) error {
		_, err := tx.Exec(query, chatID, coin, quantity)
		if err != nil {
			return fmt.Errorf("failed to create chat coins for chat %s, coin %s: %w", chatID, coin, err)
		}
		return nil
	})
}

func (c *client) GetChatCoinInfo(chatID string) ([]*CoinInfo, error) {
	var coins []*CoinInfo
	query := `SELECT coin, quantity FROM chat_coins WHERE chat_id = $1`
	err := c.Select(&coins, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get coins info for chat %s: %w", chatID, err)
	}
	return coins, nil
}

// returns empty string if not found
func (c *client) GetTicker(inputTicker string) (foundTicker string, err error) {
	query := `SELECT ticker FROM coin where ticker = $1`
	err = c.Get(&foundTicker, query, inputTicker)
	if err != nil {
		return "", fmt.Errorf("failed to get ticker %s: %w", inputTicker, err)
	}
	return foundTicker, nil
}

func (c *client) AddTickerWithQuantityToChat(chatID string, coin string, quantity decimal.Decimal) error {
	query := `
	INSERT INTO chat_coins (id, chat_id, coin, quantity)
	VALUES (gen_random_uuid(), $1, $2, $3)
	ON CONFLICT (chat_id, coin) DO UPDATE SET
	quantity = EXCLUDED.quantity;
	`
	return c.SafeTx(func(tx *sqlx.Tx) error {
		_, err := tx.Exec(query, chatID, coin, quantity)
		if err != nil {
			return fmt.Errorf("failed to add ticker %s to chat %s: %w", coin, chatID, err)
		}
		return nil
	})
}

func (c *client) RemoveCoinFromChat(chatID string, coin string) error {
	query := `DELETE FROM chat_coins WHERE chat_id = $1 AND coin = $2`
	return c.SafeTx(func(tx *sqlx.Tx) error {
		_, err := tx.Exec(query, chatID, coin)
		if err != nil {
			return fmt.Errorf("failed to remove coin %s from chat %s: %w", coin, chatID, err)
		}
		return nil
	})
}
