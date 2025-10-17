package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/smart-contract-event-indexer/shared/config"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// DB wraps sql.DB with additional functionality
type DB struct {
	*sql.DB
	logger utils.Logger
}

// NewDB creates a new database connection
func NewDB(cfg config.DatabaseConfig, logger utils.Logger) (*DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, utils.WrapError(utils.ErrCodeDatabaseConnection, "failed to open database", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, utils.WrapError(utils.ErrCodeDatabaseConnection, "failed to ping database", err)
	}

	logger.Info("Database connection established")

	return &DB{
		DB:     db,
		logger: logger,
	}, nil
}

// HealthCheck performs a database health check
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return utils.WrapError(utils.ErrCodeDatabaseConnection, "database health check failed", err)
	}

	return nil
}

// Transaction executes a function within a database transaction
func (db *DB) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return utils.WrapError(utils.ErrCodeDatabase, "failed to begin transaction", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			db.logger.WithError(rbErr).Error("Failed to rollback transaction")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return utils.WrapError(utils.ErrCodeDatabase, "failed to commit transaction", err)
	}

	return nil
}

// QueryBuilder helps build dynamic queries
type QueryBuilder struct {
	query string
	args  []interface{}
	count int
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(baseQuery string) *QueryBuilder {
	return &QueryBuilder{
		query: baseQuery,
		args:  make([]interface{}, 0),
		count: 0,
	}
}

// AddCondition adds a WHERE condition
func (qb *QueryBuilder) AddCondition(condition string, arg interface{}) *QueryBuilder {
	if qb.count == 0 {
		qb.query += " WHERE " + condition
	} else {
		qb.query += " AND " + condition
	}
	qb.args = append(qb.args, arg)
	qb.count++
	return qb
}

// AddOptionalCondition adds a condition only if the value is not nil
func (qb *QueryBuilder) AddOptionalCondition(condition string, arg interface{}) *QueryBuilder {
	if arg == nil {
		return qb
	}
	return qb.AddCondition(condition, arg)
}

// AddOrderBy adds an ORDER BY clause
func (qb *QueryBuilder) AddOrderBy(orderBy string) *QueryBuilder {
	qb.query += " ORDER BY " + orderBy
	return qb
}

// AddLimit adds a LIMIT clause
func (qb *QueryBuilder) AddLimit(limit int) *QueryBuilder {
	qb.query += fmt.Sprintf(" LIMIT %d", limit)
	return qb
}

// AddOffset adds an OFFSET clause
func (qb *QueryBuilder) AddOffset(offset int) *QueryBuilder {
	qb.query += fmt.Sprintf(" OFFSET %d", offset)
	return qb
}

// Build returns the final query and arguments
func (qb *QueryBuilder) Build() (string, []interface{}) {
	return qb.query, qb.args
}

// GetQuery returns the query string
func (qb *QueryBuilder) GetQuery() string {
	return qb.query
}

// GetArgs returns the arguments
func (qb *QueryBuilder) GetArgs() []interface{} {
	return qb.args
}

// Stats returns database statistics
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// Close closes the database connection
func (db *DB) Close() error {
	db.logger.Info("Closing database connection")
	return db.DB.Close()
}

