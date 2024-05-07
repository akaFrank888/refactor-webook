package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
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
	addr := viper.Get("redis.addr").(string)
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}
