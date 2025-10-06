package postgres

import (
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Repository interface {
	CreateDatabase(data DatabaseStorage) error
	GetDatabase(id string) (*DatabaseStorage, error)
	CountPagesByDatabase(databaseID string) (count int, err error)
	GetPagesByDatabase(databaseID string) ([]*Page, error)
	InsertTickerPage(ID, databaseID, name string) error
	GetPagesWithEmptyOrExpiredPrice(expirationTime time.Time) ([]*Page, error)
	InsertPages(pages []*Page) error
	InsertPrice(price *Price) error
	GetLastHourPriceBySymbol(symbol string) (price *Price, err error)
	GetTheLastUpdateDate() (time.Time, error)
	GetTickers() ([]string, error)
	GetLastHourUpdateBySymbol(symbol string) (time.Time, error)
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
	mu  sync.Mutex
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
		sync.Mutex{},
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

func (c *client) CreateDatabase(data DatabaseStorage) error {
	query := `
        INSERT INTO notion_databases (id, status, created_at, updated_at)
        VALUES (:id, :status, :created_at, :updated_at)
    `
	_, err := c.NamedExec(query, data)
	if err != nil {
		c.log.Error("Failed to create db", zap.Error(err))
	}
	return err
}

// GetDatabase returns the latest database by id
func (c *client) GetDatabase(id string) (*DatabaseStorage, error) {
	var database []*DatabaseStorage
	query := `SELECT * FROM notion_databases WHERE id = $1 ORDER BY created_at DESC LIMIT 1;`
	err := c.Select(&database, query, id)
	if err != nil {
		c.log.Warn("Failed to get database", zap.Error(err))
		return nil, err
	}
	return database[0], nil
}

func (c *client) CountPagesByDatabase(databaseID string) (count int, err error) {
	query := `SELECT COUNT(*) FROM pages WHERE database_id = $1`
	err = c.Get(&count, query, databaseID)
	if err != nil {
		c.log.Error("Failed to count pages by database", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// TODO: Make in batches somehow
func (c *client) GetPagesWithEmptyOrExpiredPrice(expirationTime time.Time) ([]*Page, error) {
	var pages []*Page
	query :=
		`
	SELECT * FROM pages WHERE last_price IS NULL OR last_price_timestamp <= $1
	`
	err := c.Select(&pages, query, expirationTime)
	if err != nil {
		c.log.Error("Failed to fetch pages with empty prices", zap.Error(err))
		return nil, err
	}
	return pages, nil
}

func (c *client) GetPagesByDatabase(databaseID string) ([]*Page, error) {
	var pages []*Page
	query := `SELECT * FROM pages WHERE database_id = $1 ORDER BY updated_at DESC`
	err := c.Select(&pages, query, &databaseID)
	if err != nil {
		c.log.Error("Failed to get pages by database", zap.Error(err))
		return nil, err
	}
	return pages, nil
}

func (c *client) InsertTickerPage(ID, databaseID, name string) error {
	query :=
		`
	INSERT INTO pages (id, database_id, ticker)
	VALUSES (:id, :database_id, :ticker)
	ON CONFLICT DO NOTHING;
	`
	_, err := c.Exec(query, ID, databaseID, name)
	if err != nil {
		c.log.Error("Failed to insert ticker page", zap.Error(err))
		return err
	}
	return nil
}

func (c *client) InsertPages(pages []*Page) error {
	for _, page := range pages {
		query := `
		INSERT INTO pages (id, database_id, ticker, name, asset_type, last_price, return_value_usd, last_price_timestamp, created_at, updated_at)
		VALUES (:id, :database_id, :ticker, :name, :asset_type, :last_price, :return_value_usd, :last_price_timestamp, :created_at, :updated_at)
		ON CONFLICT (id) DO UPDATE SET
		  last_price = EXCLUDED.last_price,
		  return_value_usd = EXCLUDED.return_value_usd,
		  last_price_timestamp = EXCLUDED.last_price_timestamp,
		  updated_at = EXCLUDED.updated_at;
	`
		_, err := c.NamedExec(query, page)
		if err != nil {
			c.log.Error("Failed to insert pages", zap.Error(err))
			return err
		}
	}
	return nil
}

func (c *client) InsertPrice(price *Price) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	query := `
		INSERT INTO one_hour_price (fromsym, tosym, last_price, ts)
		VALUES (:fromsym, :tosym, :last_price, :ts)
		ON CONFLICT DO NOTHING;
	`
	_, err := c.NamedExec(query, price)
	if err != nil {
		c.log.Error("Failed to insert one hour price", zap.Error(err))
		return err
	}
	return nil
}

func (c *client) GetLastHourPriceBySymbol(symbol string) (price *Price, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var prices []*Price
	query := `SELECT fromsym, tosym, last_price, ts FROM one_hour_price WHERE fromsym = $1 ORDER BY ts DESC LIMIT 1`
	err = c.Select(&prices, query, symbol)
	if err != nil || len(prices) == 0 {
		c.log.Sugar().Warnf("Failed to get last hour price for $symbol", symbol)
		return nil, err
	}
	return prices[0], nil
}

// TODO: delete?
func (c *client) GetTheLastUpdateDate() (time.Time, error) {
	var lastUpdate time.Time
	query := `SELECT MAX(ts) FROM one_hour_price`
	err := c.Get(&lastUpdate, query)
	if err != nil {
		c.log.Error("Failed to get the last update date", zap.Error(err))
		return time.Time{}, err
	}
	return lastUpdate, nil
}

func (c *client) GetTickers() ([]string, error) {
	var tickers []string
	query := `SELECT DISTINCT ticker FROM coin ORDER BY ticker`
	err := c.Select(&tickers, query)
	if err != nil {
		c.log.Error("Failed to get tickers", zap.Error(err))
		return nil, err
	}
	return tickers, nil
}

// to delete
func (c *client) GetLastHourUpdateBySymbol(symbol string) (time.Time, error) {
	var lastUpdateTime time.Time
	query := `SELECT ts FROM one_hour_price WHERE fromsym = $1 ORDER BY ts DESC LIMIT 1`
	err := c.Get(&lastUpdateTime, query, symbol)
	if err != nil {
		c.log.Error("Failed to get last hour update by symbol", zap.Error(err))
		return time.Time{}, err
	}
	return lastUpdateTime, nil
}

func (c *client) CreateChat(chatID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	query := `
		INSERT INTO tg_chat (chatId, created_at, updated_at, prime, is_active)
		VALUES ($1, NOW(), NOW(), FALSE, TRUE)
		ON CONFLICT (chatId) DO UPDATE SET 
			updated_at = NOW();
	`
	_, err := c.Exec(query, chatID)
	if err != nil {
		c.log.Error("Failed to create chat instance", zap.Error(err))
		return err
	}
	return nil
}

func (c *client) CreateChatCoins(chatID string, coin string, quantity decimal.Decimal) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	query := `
			INSERT INTO chat_coins (id, chat_id, coin, quantity) 
			VALUES (gen_random_uuid(), $1, $2, $3)
			ON CONFLICT (chat_id, coin) DO UPDATE SET
			quantity = EXCLUDED.quantity;
		`
	_, err := c.Exec(query, chatID, coin, quantity)
	if err != nil {
		c.log.Error("Failed to add coin to chat", zap.Error(err))
		return err
	}
	return nil
}

func (c *client) GetChatCoinInfo(chatID string) ([]*CoinInfo, error) {
	var coins []*CoinInfo
	query := `SELECT coin, quantity FROM chat_coins WHERE chat_id = $1`
	err := c.Select(&coins, query, chatID)
	if err != nil {
		c.log.Error("Failed to get coins info by chatID", zap.Error(err))
		return nil, err
	}
	return coins, nil
}

// returns empty string if not found
func (c *client) GetTicker(inputTicker string) (foundTicker string, err error) {
	query := `SELECT ticker FROM coin where ticker = $1`
	err = c.Get(&foundTicker, query, inputTicker)
	if err != nil {
		c.log.Sugar().Warnf("Ticker %s not found", inputTicker)
		return "", err
	}
	return foundTicker, nil
}

func (c *client) AddTickerWithQuantityToChat(chatID string, coin string, quantity decimal.Decimal) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	query := `
	INSERT INTO chat_coins (id, chat_id, coin, quantity)
	VALUES (gen_random_uuid(), $1, $2, $3)
	ON CONFLICT (chat_id, coin) DO UPDATE SET
	quantity = EXCLUDED.quantity;
	`
	_, err := c.Exec(query, chatID, coin, quantity)
	return err
}

func (c *client) RemoveCoinFromChat(chatID string, coin string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	query := `DELETE FROM chat_coins WHERE chat_id = $1 AND coin = $2`
	_, err := c.Exec(query, chatID, coin)
	return err
}
