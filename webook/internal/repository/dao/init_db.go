package dao

import (
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {

	// 配置文件名称和类型
	viper.SetConfigName("dev")
	viper.SetConfigType("yaml")
	// 当前工作目录（Working Directory）的子目录是config
	viper.AddConfigPath("config")
	// 读取配置
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	dsn := viper.Get("db.dsn").(string)
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic("failed to connect mysql database")
	}

	// 迁移 schema
	err = db.AutoMigrate(&User{})
	if err != nil {
		panic("failed to migrate database")
	}
	return db
}
