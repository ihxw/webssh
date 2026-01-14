package models

import (
	"time"

	"gorm.io/gorm"
)

type SSHHost struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	UserID              uint           `gorm:"not null;index" json:"user_id"`
	Name                string         `gorm:"size:100;not null" json:"name"`
	Host                string         `gorm:"size:255;not null" json:"host"`
	Port                int            `gorm:"default:22" json:"port"`
	Username            string         `gorm:"size:100;not null" json:"username"`
	AuthType            string         `gorm:"size:20;not null" json:"auth_type"` // password or key
	Fingerprint         string         `gorm:"size:255" json:"fingerprint"`       // SSH Host Key Fingerprint (TOFU)
	PasswordEncrypted   string         `gorm:"type:text" json:"-"`
	PrivateKeyEncrypted string         `gorm:"type:text" json:"-"`
	GroupName           string         `gorm:"size:50" json:"group_name"`
	Tags                string         `gorm:"size:255" json:"tags"`
	Description         string         `gorm:"type:text" json:"description"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`

	// Transient fields (not stored in database)
	Password   string `gorm:"-" json:"password,omitempty"`
	PrivateKey string `gorm:"-" json:"private_key,omitempty"`
}

// TableName specifies the table name
func (SSHHost) TableName() string {
	return "ssh_hosts"
}
