package mysql

import (
	"os"
	"testing"
	"time"

	"github.com/Fs02/go-paranoid"

	"github.com/Fs02/grimoire"
	. "github.com/Fs02/grimoire/c"
	"github.com/Fs02/grimoire/changeset"
	"github.com/stretchr/testify/assert"
)

type Address struct {
	ID        int64
	UserID    int64
	Address   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID        int64
	Name      string
	Gender    string
	Age       int
	Note      *string
	Addresses []Address
	CreatedAt time.Time
	UpdatedAt time.Time
}

// User table identifiers
const (
	users     = "users"
	addresses = "addresses"
	id        = I("id")
	name      = I("name")
	gender    = I("gender")
	age       = I("age")
	note      = I("note")
	createdAt = I("created_at")
	address   = I("address")
)

func init() {
	adapter, err := Open(dsn() + "?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	defer adapter.Close()

	_, _, err = adapter.Exec(`DROP TABLE IF EXISTS addresses;`, []interface{}{})
	paranoid.Panic(err)
	_, _, err = adapter.Exec(`DROP TABLE IF EXISTS users;`, []interface{}{})
	paranoid.Panic(err)

	_, _, err = adapter.Exec(`CREATE TABLE users (
		id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(30) NOT NULL,
		gender VARCHAR(10) NOT NULL,
		age INT NOT NULL,
		note varchar(50),
		created_at DATETIME,
		updated_at DATETIME
	);`, []interface{}{})
	paranoid.Panic(err)

	_, _, err = adapter.Exec(`CREATE TABLE addresses (
		id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
		user_id INT UNSIGNED,
		address VARCHAR(60) NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`, []interface{}{})
	paranoid.Panic(err)
}

func dsn() string {
	if os.Getenv("MYSQL_DATABASE") != "" {
		return os.Getenv("MYSQL_DATABASE")
	}

	return "root@(127.0.0.1:3306)/grimoire_test"
}

func changeUser(user interface{}, params map[string]interface{}) *changeset.Changeset {
	ch := changeset.Cast(user, params, []string{
		"name",
		"gender",
		"age",
		"note",
	})
	changeset.CastAssoc(ch, "addresses", changeAddress)
	return ch
}

func changeAddress(address interface{}, params map[string]interface{}) *changeset.Changeset {
	ch := changeset.Cast(address, params, []string{"address"})
	return ch
}

func TestRepoTransaction(t *testing.T) {
	adapter, err := Open(dsn() + "?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	defer adapter.Close()
	repo := grimoire.New(adapter)

	user := User{}

	params := map[string]interface{}{
		"name":   "whiteviolet",
		"gender": "male",
		"age":    18,
		"note":   "some note here",
		"addresses": []map[string]interface{}{
			{
				"address": "Aceh, Indonesia",
			},
			{
				"address": "Bandung, Indonesia",
			},
		},
	}

	ch := changeUser(user, params)
	assert.Nil(t, ch.Error())

	err = repo.Transaction(func(repo grimoire.Repo) error {
		repo.From("users").MustInsert(&user, ch)
		addresses := ch.Changes()["addresses"].([]*changeset.Changeset)
		repo.From("addresses").Set("user_id", user.ID).MustInsert(&user.Addresses, addresses...)

		return nil
	})

	assert.Nil(t, err)
	assert.True(t, user.ID > 0)
	assert.Equal(t, "whiteviolet", user.Name)
	assert.Equal(t, "male", user.Gender)
	assert.Equal(t, 18, user.Age)
	assert.Equal(t, "some note here", *user.Note)
	assert.NotEqual(t, time.Time{}, user.CreatedAt)
	assert.NotEqual(t, time.Time{}, user.UpdatedAt)

	assert.Equal(t, 2, len(user.Addresses))
	assert.Equal(t, "Aceh, Indonesia", user.Addresses[0].Address)
	assert.NotEqual(t, time.Time{}, user.Addresses[0].CreatedAt)
	assert.NotEqual(t, time.Time{}, user.Addresses[0].UpdatedAt)
	assert.Equal(t, "Bandung, Indonesia", user.Addresses[1].Address)
	assert.NotEqual(t, time.Time{}, user.Addresses[1].CreatedAt)
	assert.NotEqual(t, time.Time{}, user.Addresses[1].UpdatedAt)
}
