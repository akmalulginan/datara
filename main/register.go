package main

import (
	"fmt"
	"os"
	"time"

	"ariga.io/atlas-provider-gorm/gormschema"
)

type User struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement"`
	Username     string     `gorm:"type:varchar(100);not null;uniqueIndex:uni_users_username"`
	Email        string     `gorm:"type:varchar(255);not null;uniqueIndex:uni_users_email"`
	Password     string     `gorm:"type:varchar(255);not null"`
	IsActive     bool       `gorm:"not null;default:true"`
	LastLoginAt  *time.Time `gorm:"type:timestamp with time zone"`
	CreatedAt    time.Time  `gorm:"type:timestamp with time zone;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time  `gorm:"type:timestamp with time zone;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt    *time.Time `gorm:"type:timestamp with time zone"`
	LastLocation string     `gorm:"type:varchar(255)"`
}

type Profile struct {
	ID          uint      `gorm:"type:bigserial;primaryKey"`
	UserID      uint      `gorm:"type:bigint;not null;references:users(id)"`
	Bio         string    `gorm:"type:varchar(500)"`
	PhoneNumber string    `gorm:"type:varchar(20)"`
	IsVerified  bool      `gorm:"type:boolean;default:false;not null"`
	CreatedAt   time.Time `gorm:"type:timestamp with time zone;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `gorm:"type:timestamp with time zone;not null;default:CURRENT_TIMESTAMP"`
	Avatar      string    `gorm:"type:varchar(255)"`
	Address     string    `gorm:"type:varchar(1000)"`
	Website     string    `gorm:"type:varchar(255)"`
	Notes       string    `gorm:"type:text"`
	Level       int       `gorm:"type:integer;default:1"`
	Experience  int64     `gorm:"type:bigint;default:0"`
	Title       string    `gorm:"type:varchar(100)"`
	Badges      string    `gorm:"type:text[]"`
	Settings    string    `gorm:"type:jsonb"`
	Metadata    string    `gorm:"type:jsonb"`
	Tags        string    `gorm:"type:text[]"`
	Status      string    `gorm:"type:varchar(50);default:'active'"`
	Score       float64   `gorm:"type:decimal(10,2);default:0"`
	Rating      float64   `gorm:"type:decimal(5,2);default:0"`
	Points      int64     `gorm:"type:bigint;default:100"`
	Balance     float64   `gorm:"type:decimal(15,4);default:0"`
	Weight      float64   `gorm:"type:decimal(8,3);default:0"`
	Height      float64   `gorm:"type:decimal(6,2);default:0"`
	Age         int       `gorm:"type:smallint;default:0"`
	Price       float64   `gorm:"type:decimal(12,2);default:0"`
	Tax         float64   `gorm:"type:decimal(8,4);default:0"`
	Discount    float64   `gorm:"type:decimal(5,2);default:0"`
	Quantity    int       `gorm:"type:integer;default:1"`
}

func main() {
	stmts, err := gormschema.New("postgres").Load(
		&User{},
		&Profile{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}

	// Output schema SQL
	fmt.Print(stmts)
}
