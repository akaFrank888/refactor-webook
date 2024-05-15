package ioc

import (
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"refactor-webook/webook/internal/repository/dao"
	"refactor-webook/webook/pkg/logger"
)

func InitDB(l logger.LoggerV1) *gorm.DB {

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
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: glogger.New(goormLoggerFunc(l.Debug), glogger.Config{
			// 慢查询
			SlowThreshold: 0,
			LogLevel:      glogger.Info,
		}),
	})
	if err != nil {
		panic("failed to connect mysql database")
	}

	// 迁移 schema
	err = dao.InitTables(db)
	if err != nil {
		panic("failed to migrate database")
	}
	return db
}

type goormLoggerFunc func(msg string, fields ...logger.Field)

func (g goormLoggerFunc) Printf(s string, i ...interface{}) {
	g(s, logger.Field{Key: "args", Val: i})
}
