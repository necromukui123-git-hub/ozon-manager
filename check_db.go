package main

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Shop struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"size:100;not null"`
	ClientID string `gorm:"size:50;uniqueIndex;not null"`
	ApiKey   string `gorm:"size:200;not null"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("backend/ozon.db"), &gorm.Config{})
	if err != nil {
		fmt.Println("failed to connect database")
		return
	}

	var shops []Shop
	db.Find(&shops)

	for _, shop := range shops {
		fmt.Printf("ID: %d, Name: %s, ClientID: '%s', ApiKey: '%s'\n", shop.ID, shop.Name, shop.ClientID, shop.ApiKey)
	}
}
