package storage

import (
	"gRPCAuth/api"
	"github.com/bradfitz/gomemcache/memcache"
	"log"
	"strings"
)

// Exist checks if user exist in the database or returns error
func Exist(storages *Storage, user *api.User) (bool, bool) { //exist, sameHash
	passwd, err := getUserPassword(storages, user.Login)
	if err != nil {
		return false, false
	}
	if passwd == user.Password {
		return true, true
	}
	return true, false
}

func getUserPassword(storages *Storage, login string) (string, error) {
	passwd, err := getFromCache(storages, login)
	if err != nil {
		passwd, err = getFromDB(storages.PSQL, login)
		if err != nil {
			if !strings.Contains(err.Error(), "sql: no rows in result set") {
				log.Printf("getUser:getFromDB: %s", err)
			}
			return "", err
		}
		if err = storages.Cache.Set(&memcache.Item{Key: login, Value: []byte(passwd), Expiration: 3600}); err != nil {
			log.Printf("getUser:Set: %s", err)
			return "", err
		}
	}
	return passwd, nil
}

func getFromCache(storages *Storage, login string) (string, error) {
	password, err := storages.Cache.Get(login)
	if err != nil {
		if err.Error() != "memcache: cache miss" {
			log.Printf("getFromCache:%s", err)
		}
		return "", err
	}
	return string(password.Value), nil
}

func getFromDB(postgres Postgres, login string) (string, error) {
	var password string
	query := "SELECT password FROM users WHERE login = $1"
	err := postgres.QueryRow(query, login).Scan(&password)
	if err != nil {
		return "", err
	}
	return password, err
}
