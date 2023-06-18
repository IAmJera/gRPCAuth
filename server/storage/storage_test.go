package storage_test

import (
	"fmt"
	"gRPCAuth/api"
	"gRPCAuth/server/storage"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

type testCache struct{}

func (c *testCache) Set(item *memcache.Item) error {
	switch item.Key {
	case "setSuccess":
		return nil
	case "setError":
		return fmt.Errorf("error")
	}
	return nil
}
func (c *testCache) Get(key string) (item *memcache.Item, err error) {
	switch key {
	case "setError":
		return nil, fmt.Errorf("memcache: cache miss")
	case "dbError":
		return nil, fmt.Errorf("error")
	case "success":
		return &memcache.Item{Key: "setSuccess", Value: []byte("password")}, nil
	case "setSuccess":
		return &memcache.Item{Key: "setSuccess", Value: []byte("wrongPassword")}, nil
	}
	return nil, nil
}
func (c *testCache) Replace(_ *memcache.Item) error {
	return nil
}
func (c *testCache) Delete(_ string) error {
	return nil
}

func TestExist(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	strg := storage.Storage{Cache: &testCache{}, PSQL: db}

	testCases := []struct {
		name  string
		user  *api.User
		bool1 bool
		bool2 bool
	}{
		{
			name:  "database error",
			user:  &api.User{Login: "dbError", Password: "password"},
			bool1: false,
			bool2: false},
		{
			name:  "set cache error",
			user:  &api.User{Login: "setError", Password: "password"},
			bool1: false,
			bool2: false},
		{
			name:  "success, wrong password",
			user:  &api.User{Login: "setSuccess", Password: "password"},
			bool1: true,
			bool2: false},
		{
			name:  "success",
			user:  &api.User{Login: "success", Password: "password"},
			bool1: true,
			bool2: true},
	}

	for _, tc := range testCases {
		query := "SELECT password FROM users WHERE login = $1"
		switch tc.user.Login {
		case "SQLUnknownError":
			mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(tc.user.Login).WillReturnError(fmt.Errorf("error"))
		case "dbError":
			mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(tc.user.Login).WillReturnError(fmt.Errorf("sql: no rows"))
		case "setSuccess":
			rows1 := sqlmock.NewRows([]string{"password"}).AddRow("wrongPassword")
			mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(tc.user.Login).WillReturnRows(rows1)
		case "success":
			fallthrough
		case "setError":
			rows2 := sqlmock.NewRows([]string{"password"}).AddRow("password")
			mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(tc.user.Login).WillReturnRows(rows2)
		}

		gotBool1, gotBool2 := storage.Exist(&strg, tc.user)
		assert.Equal(t, gotBool1, tc.bool1)
		assert.Equal(t, gotBool2, tc.bool2)
	}
}
