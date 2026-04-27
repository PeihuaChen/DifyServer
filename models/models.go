package models

import (
	"time"
)

type Account struct {
	ID                string `json:"ID"`
	Name              string `json:"Name"`
	Email             string `json:"Email"`
	Password          string `json:"Password"`
	PasswordSalt      string `json:"PasswordSalt"`
	Avatar            string `json:"Avatar"`
	InterfaceLanguage string `json:"InterfaceLanguage"`
	InterfaceTheme    string `json:"InterfaceTheme"`
	Timezone          string `json:"Timezone"`
	Status            string `json:"Status"`
}

type Tenant struct {
	ID               string `gorm:"primaryKey"`
	Name             string
	EncryptPublicKey string
	Plan             string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CustomConfig     *string
}

type TenantAccountJoin struct {
	ID        string `gorm:"primaryKey"`
	TenantID  string
	AccountID string
	Role      string
	InvitedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
	Current   bool
}

type Dataset struct {
	ID                     string `gorm:"primaryKey"`
	TenantID               string
	Name                   string
	Description            string
	Provider               string
	Permission             string
	DataSourceType         string
	IndexingTechnique      string
	IndexStruct            string
	CreatedBy              string
	CreatedAt              time.Time
	UpdatedBy              string
	UpdatedAt              time.Time
	EmbeddingModel         string
	EmbeddingModelProvider string
	CollectionBindingID    string
	RetrievalModel         string
}

// PluginInstallation Dify 插件安装记录（只读，用于查询已安装插件）
type PluginInstallation struct {
	ID                     string `gorm:"primaryKey" json:"id"`
	TenantID               string `json:"tenant_id"`
	PluginID               string `json:"plugin_id"`
	PluginUniqueIdentifier string `json:"plugin_unique_identifier"`
	Source                 string `json:"source"`
}

func (PluginInstallation) TableName() string {
	return "plugin_installations"
}
