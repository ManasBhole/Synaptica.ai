package deid

import "time"

const (
	DefaultAnonymity = "k-3"
)

type TokenRecord struct {
	Token     string    `gorm:"primaryKey;column:token" json:"token"`
	Value     string    `gorm:"column:value" json:"value"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (TokenRecord) TableName() string {
	return "deid_token_vault"
}
