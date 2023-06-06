package storage

import (
	"database/sql"
	"fmt"
	"github.com/bradfitz/gomemcache/memcache"
	_ "github.com/lib/pq"
	"log"
	"os"
	"strings"
	"time"
)

type Storage struct {
	Cache *memcache.Client
	PSQL  *sql.DB
}

// InitStorages initializes all storages and returns the structure with them
func InitStorages() *Storage {
	store := Storage{}
	store.Cache = memcache.New(os.Getenv("CACHE_ADDRESS"))
	var err error
	if store.PSQL, err = initPSQL(); err != nil {
		log.Fatal(err)
	}

	exist, err := TableExist(store.PSQL)
	if err != nil {
		log.Panicf("InitStorages:TableExist: %s", err)
	}
	if !exist {
		if err = CreateTable(store.PSQL); err != nil {
			log.Panicf("InitStorages:CreateTable: %s", err)
		}
	}
	return &store
}

func initPSQL() (*sql.DB, error) {
	addr := os.Getenv("PSQL_ADDRESS")
	port := os.Getenv("PSQL_PORT")
	login := os.Getenv("POSTGRES_USER")
	passwd := os.Getenv("POSTGRES_PASSWORD")
	database := os.Getenv("POSTGRES_DB")
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		addr, port, login, passwd, database)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, err
}

func TableExist(db *sql.DB) (bool, error) {
	if _, err := db.Query("SELECT * FROM " + os.Getenv("POSTGRES_DB") + ";"); err != nil {
		if strings.Contains(err.Error(), "pq: relation \"users\" does not exist") {
			return false, nil
		}
		log.Printf("tableExist:Query: %s", err)
		return false, err
	}
	return true, nil
}

func CreateTable(db *sql.DB) error {
	query := "CREATE TABLE " + os.Getenv("POSTGRES_DB") +
		" ( login VARCHAR(30) UNIQUE NOT NULL, password VARCHAR (64) NOT NULL);"
	if _, err := db.Query(query); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		log.Printf("prepareDB:Query: %s", err)
		return err
	}
	return nil
}
