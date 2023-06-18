package auth

import (
	"context"
	"gRPCAuth/api"
	"gRPCAuth/server/storage"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/golang-jwt/jwt"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"log"
	"os"
	"time"
)

// User defines the user structure
type User struct {
	jwt.StandardClaims
	Login    string
	Password string
	Role     string
}

// GRPCServer is the instance of AuthServer interface
type GRPCServer struct {
	api.UnimplementedAuthServer
}

var storages = storage.InitStorages()

// UserExist returns error if user does not exist
func (s *GRPCServer) UserExist(_ context.Context, user *api.User) (*wrapperspb.BoolValue, error) {
	if exist, _ := storage.Exist(storages, user); exist {
		return &wrapperspb.BoolValue{Value: true}, nil
	}
	return &wrapperspb.BoolValue{Value: false}, nil
}

// AddUser adds user to the database or returns error
func (s *GRPCServer) AddUser(_ context.Context, user *api.User) (*wrapperspb.StringValue, error) {
	if exist, _ := storage.Exist(storages, user); exist {
		return &wrapperspb.StringValue{Value: "user already exist"}, nil
	}

	query := "INSERT INTO users (login, password) VALUES ($1, $2)"
	if _, err := storages.PSQL.Query(query, user.Login, user.Password); err != nil {
		log.Printf("AddUser:Query: %s", err)
		return &wrapperspb.StringValue{}, err
	}

	err := storages.Cache.Set(&memcache.Item{Key: user.Login, Value: []byte(user.Password), Expiration: 3600})
	if err != nil {
		log.Printf("AddUser:Set: %s", err)
	}
	return &wrapperspb.StringValue{}, err
}

// UpdateUser updates user credentials
func (s *GRPCServer) UpdateUser(_ context.Context, user *api.User) (*wrapperspb.StringValue, error) {
	if exist, _ := storage.Exist(storages, user); !exist {
		return &wrapperspb.StringValue{Value: "user does not exist"}, nil
	}

	query := "UPDATE users SET password = $2 WHERE login = $1"
	if _, err := storages.PSQL.Query(query, user.Login, user.Password); err != nil {
		log.Printf("UpdateUser:Query: %s", err)
		return &wrapperspb.StringValue{}, err
	}

	err := storages.Cache.Replace(&memcache.Item{Key: user.Login, Value: []byte(user.Password), Expiration: 3600})
	if err != nil {
		log.Printf("UpdateUser:Replace: %s", err)
	}
	return &wrapperspb.StringValue{}, err
}

// DelUser removes user from storages
func (s *GRPCServer) DelUser(_ context.Context, user *api.User) (*wrapperspb.StringValue, error) {
	if exist, _ := storage.Exist(storages, user); !exist {
		return &wrapperspb.StringValue{Value: "user does not exist"}, nil
	}

	query := "DELETE FROM users WHERE login = $1"
	if _, err := storages.PSQL.Query(query, user.Login); err != nil {
		log.Printf("DelUser:Query: %s", err)
		return &wrapperspb.StringValue{}, err
	}

	if err := storages.Cache.Delete(user.Login); err != nil {
		if err.Error() != "memcache: cache miss" {
			log.Printf("DelUser:Delete: %s", err)
			return &wrapperspb.StringValue{}, err
		}
	}
	return &wrapperspb.StringValue{}, nil
}

// GetToken returns token of the user of error
func (s *GRPCServer) GetToken(_ context.Context, user *api.User) (*api.Token, error) {
	exist, sameHash := storage.Exist(storages, user)
	if !exist {
		return &api.Token{Error: "user does not exist"}, nil
	}
	if sameHash {
		token, err := createToken(user)
		if err != nil {
			log.Printf("SignIn:getToken: %s", err)
			return &api.Token{}, err
		}
		return &api.Token{Token: token}, nil
	}
	return &api.Token{Error: "incorrect password"}, nil
}

func createToken(user *api.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &User{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		Login: user.Login,
		Role:  user.Role,
	})
	return token.SignedString([]byte(os.Getenv("SIGNING_KEY")))
}
