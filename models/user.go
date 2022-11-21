package models

import (
	"errors"
	"strings"
	"time"

	"encoding/json"
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	gorm.Model
	ID        uint32    `gorm:"primary_key;auto_increment" json:"id"`
	Username  string    `gorm:"size:255;not null;unique" json:"username"`
	Password  string    `gorm:"size:100;not null;" json:"password"`
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type UserNodes struct {
	UserNodes []User `json:"user_nodes"`
}

func JsonSeed() {
	file, _ := ioutil.ReadFile("users.json")
	data := UserNodes{}

	_ = json.Unmarshal([]byte(file), &data)

	for idx := 0; idx < len(data.UserNodes); idx++ {
		fmt.Println("User: ", data.UserNodes[idx].Username)

	}
}

func AllUsers() []User {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		println(err.Error())
		panic("failed to connect database")
	}

	var users []User
	db.Find(&users)

	return users
}

func GetUserByUsername(username string) User {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		println(err.Error())
		panic("failed to connect database")
	}

	var user User
	db.Where(User{Username: username}).First(&user)

	return user
}

func InserUser(u *User) {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		println(err.Error())
		panic("failed to connect database")
	}

	// Upsert -> Update only password.
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "username"}},
		DoUpdates: clause.AssignmentColumns([]string{"password"}),
	}).Create(&u)
}

func BulkInsertUser(u *[]User) {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		println(err.Error())
		panic("failed to connect database")
	}

	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "username"}},
		DoUpdates: clause.AssignmentColumns([]string{"password"}),
	}).Create(&u)
}

func Hash(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// func (u *User) BeforeSave(db *gorm.DB) error {
// 	hasshedPassword, err := Hash(u.Password)
// 	if err != nil {
// 		return err
// 	}

// 	u.Password = string(hasshedPassword)
// 	return nil
// }

func (u *User) Prepare() {
	u.ID = 0
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
}

func (u *User) Validate(action string) error {
	switch strings.ToLower(action) {
	case "login":
		if u.Password == "" {
			return errors.New("required Password")
		}
		return nil
	default:
		// Create case
		if u.Username == "" {
			return errors.New("required Username")
		}
		if u.Password == "" {
			return errors.New("required Password")
		}
		return nil
	}
}

func (u *User) SaveUser(db *gorm.DB) (*User, error) {
	var err = db.Debug().Create(&u).Error

	if err != nil {
		return &User{}, err
	}

	return u, nil
}

func (u *User) FindAllUsers(db *gorm.DB) (*[]User, error) {
	users := []User{}
	var err = db.Debug().Model(&User{}).Find(&users).Error

	if err != nil {
		return &[]User{}, err
	}

	return &users, err
}

func (u *User) FindUserByUsername(db *gorm.DB, username string) (*User, error) {
	var err = db.Debug().Where(&User{Username: username}).Take(&u).Error

	if err != nil {
		return &User{}, err
	}

	return u, err
}

func (u *User) UpdateAUser(db *gorm.DB, username string) (*User, error) {

	// To hash the password
	// err := u.BeforeSave(db)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	db = db.Debug().Model(&User{}).Where(&User{Username: username}).Take(&User{}).UpdateColumns(
		map[string]interface{}{
			"password":  u.Password,
			"update_at": time.Now(),
		},
	)

	if db.Error != nil {
		return &User{}, db.Error
	}

	// This is the display the updated user
	err := db.Debug().Model(&User{}).Where(&User{Username: username}).Take(&u).Error

	if err != nil {
		return &User{}, err
	}

	return u, nil
}

func (u *User) DeleteAUser(db *gorm.DB, username string) (int64, error) {
	db = db.Debug().Model(&User{}).Where(&User{Username: username}).Take(&User{}).Delete(&User{})

	if db.Error != nil {
		return 0, db.Error
	}

	return db.RowsAffected, nil
}

// Realm.

type Realm struct {
	gorm.Model
	Name string
}

type UserRealm struct {
	// User <> Realm
	gorm.Model
	User  User
	Realm Realm
}
