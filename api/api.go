package api

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lwmacct/250300-go-mod-mlog/pkg/mlog"
	"github.com/lwmacct/250402-m-geoip/api/v10/models"
	"github.com/lwmacct/250402-m-geoip/app"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mux struct {
	router *gin.Engine
}

func (m *mux) InitDb(dsn string) *mux {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Error),
		Logger:                 Logger{}.LogMode(logger.Error),
		SkipDefaultTransaction: true, // 禁用默认事务
	})
	if err != nil {
		mlog.Error(mlog.H{"msg": "api.InitDb", "err": err})
		return m
	}

	geoip := &models.GeoIPV10{}

	// 自动迁移GeoIP模型
	if err := db.AutoMigrate(geoip); err != nil {
		mlog.Error(mlog.H{"msg": "api.InitDb", "err": err, "detail": "AutoMigrate failed"})
	} else {
		mlog.Info(mlog.H{"msg": "api.InitDb", "detail": "AutoMigrate successful"})
	}

	// 使用TableIndex方法创建索引
	if err := geoip.TableIndex(db); err != nil {
		mlog.Error(mlog.H{"msg": "api.InitDb", "err": err, "detail": "Failed to create indexes"})
	} else {
		mlog.Info(mlog.H{"msg": "api.InitDb", "detail": "Indexes created successfully"})
	}

	app.DB = db
	return m
}

func New() *mux {
	// gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.Use(gin.Recovery())
	return &mux{router: r}
}

func (t *mux) register() {
	new(routerV10).Init(t.router.Group("/api/v10"))
}

func (t *mux) Run() {
	// 检查数据库连接
	if app.DB == nil {
		mlog.Error(mlog.H{"msg": "api.Run", "error": "database connection not initialized, please call InitDb first"})
		return
	}

	{
		// 判断数据库是否为空,如果为空则导入IP地理位置数据
		record := &models.GeoIPV10{}
		if err := app.DB.Take(record).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			mlog.Info(mlog.H{"msg": "api.Run", "data": "InsertGeoIP"})
			record.InsertSampleData(app.DB)
		}
	}

	// 注册路由并启动服务器
	t.register()
	mlog.Info(mlog.H{"msg": "api.Run", "data": "Server starting on " + app.Flag.App.ListenAddr})
	if err := t.router.Run(app.Flag.App.ListenAddr); err != nil {
		mlog.Error(mlog.H{"msg": "api.Run", "err": err})
	}
	mlog.Info(mlog.H{"msg": "api.Run", "data": "Run completed"})
}

func (t *mux) Test() {

}
