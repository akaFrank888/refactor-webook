package ioc

import (
	prometheus2 "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/prometheus"
	"refactor-webook/webook/internal/repository/dao"
	"refactor-webook/webook/pkg/gormx"
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
		//Logger: glogger.New(goormLoggerFunc(l.Debug), glogger.Config{
		//	// 慢查询
		//	SlowThreshold: 0,
		//	LogLevel:      glogger.Info,
		//}),
	})
	if err != nil {
		panic("failed to connect mysql database")
	}

	// note 接入 GORM 自己的 prometheus
	err = db.Use(prometheus.New(prometheus.Config{
		DBName: "webook",
		// note 每 15s 收集一些数据
		RefreshInterval: 15,
		MetricsCollector: []prometheus.MetricsCollector{
			&prometheus.MySQL{
				VariableNames: []string{"thread_running"},
			},
		},
	}))
	if err != nil {
		panic(err)
	}

	cb := gormx.NewCallbacks(prometheus2.SummaryOpts{
		Namespace: "webook",
		Subsystem: "gorm",
		Name:      "gorm_db",
		Help:      "统计 GORM 的数据库查询",
		ConstLabels: map[string]string{
			"instance_id": "my_instance",
		},
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.75:  0.01,
			0.9:   0.01,
			0.99:  0.001,
			0.999: 0.0001,
		},
	})

	err = db.Use(cb)
	if err != nil {
		panic(err)
	}

	// note 建表（迁移 schema）
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
