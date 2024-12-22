package main

import (
	"fmt"
	"time"

	"github.com/akmalulginan/datara"
)

type User struct {
	ID          uint `db:"type=primary_key,auto_increment"`
	Username    string
	Email       string
	Password    string
	PhoneNumber string
	Address     string
	Status      string
	Role        string
	IsActive    bool
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type Profile struct {
	ID         uint `db:"type=primary_key,auto_increment"`
	UserID     uint
	Bio        string
	Avatar     string
	Location   string
	Website    string
	Phone      string
	IsVerified bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func main() {
	// Buat skema baru
	schema := datara.ParseSchema(
		User{},
		Profile{},
	)

	// Output SQL
	fmt.Print(schema.ToSQL())
}
