package auth

import (
	"context"
	"fmt"
	"gRPCAuth/api"
	"gRPCAuth/server/storage"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/golang-jwt/jwt"
	"log"
	"os"
	"strings"
	"time"
)

// GRPCServer is the instance of AuthServer interface
type GRPCServer struct {
	api.UnimplementedAuthServer
}

var storages = storage.InitStorages()

// AddUser adds the user to the database or returns error
func (s *GRPCServer) AddUser(_ context.Context, user *api.User) (*api.Response, error) {
	query := "INSERT INTO " + os.Getenv("POSTGRES_DB") + " (login, password) VALUES ($1, $2)"
	if _, err := storages.PSQL.Query(query, user.Login, user.Password); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint \"users_login_key\"") {
			return nil, fmt.Errorf("user already exist")
		}
		return &api.Response{}, err
	}

	err := storages.Cache.Set(&memcache.Item{Key: user.Login, Value: []byte(user.Password)})
	if err != nil {
		log.Printf("SetUser:Set: %s", err)
	}
	return &api.Response{}, err
}

// SignIn returns token of the user of error
func (s *GRPCServer) SignIn(_ context.Context, user *api.User) (*api.Token, error) {
	exist, sameHash := storage.Exist(storages, user)
	if !exist {
		return &api.Token{}, fmt.Errorf("user does not exist")
	}
	if sameHash {
		token, err := getToken(user)
		if err != nil {
			return &api.Token{}, err
		}
		return &api.Token{Token: token}, nil
	}
	return &api.Token{}, fmt.Errorf("incorrect password")
}

func getToken(user *api.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &storage.User{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		Login: user.Login,
		Role:  user.Role,
	})
	return token.SignedString([]byte(os.Getenv("SIGNINGKEY")))
}
