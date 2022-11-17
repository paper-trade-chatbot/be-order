package database

import (
	"context"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	// MySQL driver.
	_ "github.com/jinzhu/gorm/dialects/mysql"
	// Google Cloud SQL MySQL driver.
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"

	"github.com/paper-trade-chatbot/be-order/logging"
)

// mysqlDB is the concrete MySQL handle to a SQL database.
type mysqlDB struct{ *gorm.DB }

func (db *mysqlDB) initialize(ctx context.Context, cfg dbConfig) {
	// Assemble MySQL connection params & host string.
	params := fmt.Sprintf("charset=utf8mb4&parseTime=True&loc=Local")
	host := fmt.Sprintf("(%s:%s)", cfg.Address, cfg.Port)
	if cfg.Dialect == "cloudsqlmysql" {
		cfg.Dialect = "mysql"
		host = fmt.Sprintf("cloudsql(%s)", cfg.Address)
	}
	dbSource := fmt.Sprintf("%s:%s@%s/%s?%s", cfg.Username, cfg.Password,
		host, cfg.DBName, params)
	logging.Info(ctx, "open %s:%s@%s/%s?%s", cfg.Username, cfg.Password,
		host, cfg.DBName, params)

	// Connect to the MySQL database.
	var err error
	db.DB, err = gorm.Open(mysql.Open(dbSource))
	logging.Info(ctx, "gorm open success")

	if err != nil {
		logging.Error(ctx, "mysql init err %v", err)
		panic(err)
	}

	db.Debug()
}

// finalize finalizes the MySQL database handle.
func (db *mysqlDB) finalize() {
	// Close the MySQL database handle.
	// if err := db.Close(); err != nil {
	// 	logging.Error(ctx, "Failed to close database handle: %v", err)
	// }
}

// db returns the MySQL GORM database handle.
func (db *mysqlDB) db() *gorm.DB {
	return db.DB
}
