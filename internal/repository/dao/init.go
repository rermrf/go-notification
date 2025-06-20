package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&BusinessConfig{},
		&Notification{},
		&TxNotification{},
		&CallbackLog{},
		&Provider{},
		&ChannelTemplate{},
		&ChannelTemplateVersion{},
		&ChannelTemplateProvider{},
		&Quota{},
	)
}
