package db

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
)

type User struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Id        string `json:"id"`
	Password  string `json:"password"`
}

type Users []User

type Db struct {
	data Users
}

func NewDb() *Db {
	db := &Db{}

	db.Open()

	return db
}

func (db *Db) Open() {
	fp, _ := filepath.Abs("./examples/data.json")
	dataFile, err := os.Open(fp)
	defer dataFile.Close()

	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("Successfully read file")

	bytes, _ := io.ReadAll(dataFile)

	var users Users
	json.Unmarshal(bytes, &users)

	log.Printf("Users unmarshaled %d", len(users))

	db.data = users
}

func (db *Db) FindUserByPassword(password string) *User {
	for _, user := range db.data {
		if user.Password == password {
			return &user
		}
	}

	return nil
}
