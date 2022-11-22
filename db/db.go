package db

import (
	"github.com/emyt-io/emyt/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

// var err error

func Init() {

	db, err := gorm.Open(sqlite.Open("emyt.db"), &gorm.Config{})

	if err != nil {
		panic("DB Connection error")
	}

	db.AutoMigrate(&models.User{})
}

func DbManager() *gorm.DB {
	return db
}
