package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// SafeTx выполняет функцию в транзакции с автоматическим rollback при ошибке
func (c *client) SafeTx(fn func(*sqlx.Tx) error) error {
	tx, err := c.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Безопасно - rollback игнорируется после commit

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// SafeTxWithIsolation выполняет функцию в транзакции с указанным уровнем изоляции
func (c *client) SafeTxWithIsolation(isolation sql.IsolationLevel, fn func(*sqlx.Tx) error) error {
	tx, err := c.BeginTxx(context.Background(), &sql.TxOptions{
		Isolation: isolation,
	})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}