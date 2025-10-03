package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	interfaces "github.com/prayog/prayog-supply-rate-service/internal/shared/interfaces/v1"
	models "github.com/prayog/prayog-supply-rate-service/internal/shared/models/v1"
)

// PostgresConfig holds PostgreSQL database configuration
type PostgresConfig struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	SSLMode         string
	TimeZone        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	LogLevel        string
}

// DefaultPostgresConfig returns default PostgreSQL configuration
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		Username:        "postgres",
		Password:        "password",
		Database:        "prayog_supply_rate_sandbox",
		SSLMode:         "disable",
		TimeZone:        "UTC",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
		LogLevel:        "info",
	}
}

// PostgresDB implements Database interface for PostgreSQL
type PostgresDB struct {
	db     *gorm.DB
	config *PostgresConfig
	logger interfaces.Logger
}

// NewPostgresDB creates a new PostgreSQL database instance
func NewPostgresDB(config *PostgresConfig, appLogger interfaces.Logger) *PostgresDB {
	return &PostgresDB{
		config: config,
		logger: appLogger,
	}
}

// Connect establishes connection to PostgreSQL database
func (p *PostgresDB) Connect() error {
	dsn := p.buildDSN()

	p.logger.Info("Connecting to PostgreSQL database",
		"host", p.config.Host,
		"port", p.config.Port,
		"database", p.config.Database)

	// Configure GORM logger
	var gormLogLevel logger.LogLevel
	switch p.config.LogLevel {
	case "silent":
		gormLogLevel = logger.Silent
	case "error":
		gormLogLevel = logger.Error
	case "warn":
		gormLogLevel = logger.Warn
	case "info":
		gormLogLevel = logger.Info
	default:
		gormLogLevel = logger.Info
	}

	gormLogger := logger.New(
		&gormLogWriter{logger: p.logger},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormLogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})

	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Get underlying *sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(p.config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(p.config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(p.config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(p.config.ConnMaxIdleTime)

	p.db = db

	p.logger.Info("Successfully connected to PostgreSQL database")
	return nil
}

// Close closes database connection
func (p *PostgresDB) Close() error {
	if p.db == nil {
		return nil
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}

	p.logger.Info("Closing PostgreSQL database connection")
	return sqlDB.Close()
}

// Migrate runs database migrations
func (p *PostgresDB) Migrate() error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	p.logger.Info("Running database migrations")

	// Auto migrate all models
	err := p.db.AutoMigrate(
		&models.Partner{},
		&models.Rate{},
		&models.RateResponse{},
		&models.RateQuery{},
		&models.UnifiedRateCard{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	p.logger.Info("Database migrations completed successfully")
	return nil
}

// GetDB returns the underlying GORM database instance
func (p *PostgresDB) GetDB() interface{} {
	return p.db
}

// Health checks database health
func (p *PostgresDB) Health() error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Transaction executes function within a database transaction
func (p *PostgresDB) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create a new context with the transaction
		txCtx := context.WithValue(ctx, "tx", tx)
		return fn(txCtx)
	})
}

// buildDSN builds the PostgreSQL connection string
func (p *PostgresDB) buildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		p.config.Host,
		p.config.Port,
		p.config.Username,
		p.config.Password,
		p.config.Database,
		p.config.SSLMode,
		p.config.TimeZone,
	)
}

// gormLogWriter adapts our logger to GORM's logger interface
type gormLogWriter struct {
	logger interfaces.Logger
}

// Printf implements GORM's logger.Writer interface
func (w *gormLogWriter) Printf(format string, v ...interface{}) {
	w.logger.Info(fmt.Sprintf(format, v...))
}

