package api

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lwmacct/250300-go-mod-mlog/pkg/mlog"
	"github.com/lwmacct/250402-m-geoip/api/v10/models"
	"github.com/lwmacct/250402-m-geoip/app"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type mux struct {
	router *gin.Engine
}

func (m *mux) InitDb(dsn string) *mux {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Error),
		SkipDefaultTransaction: true, // 禁用默认事务
	})
	if err != nil {
		mlog.Error(mlog.H{"msg": "api.InitDb", "err": err})
		return m
	}

	// 自动迁移GeoIP模型
	if err := db.AutoMigrate(&models.GeoIPV10{}); err != nil {
		mlog.Error(mlog.H{"msg": "api.InitDb", "err": err, "detail": "AutoMigrate failed"})
	} else {
		mlog.Info(mlog.H{"msg": "api.InitDb", "detail": "AutoMigrate successful"})
	}

	// 使用TableIndex方法创建索引
	geoip := models.GeoIPV10{}
	if err := geoip.TableIndex(db); err != nil {
		mlog.Error(mlog.H{"msg": "api.InitDb", "err": err, "detail": "Failed to create indexes"})
	} else {
		mlog.Info(mlog.H{"msg": "api.InitDb", "detail": "Indexes created successfully"})
	}

	// 收集数据库状态
	var tableCount int64
	db.Table(geoip.TableName()).Count(&tableCount)
	mlog.Info(mlog.H{"msg": "api.InitDb", "detail": "Database initialized", "records": tableCount})
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

	// 先导入IP地理位置数据
	mlog.Info(mlog.H{"msg": "api.Run", "data": "Starting InsertGeoIP"})
	{
		// 判断数据库是否为空
		var count int64
		app.DB.Model(&models.GeoIPV10{}).Count(&count)
		if count == 0 {
			t.InsertGeoIP()
		}
	}

	mlog.Info(mlog.H{"msg": "api.Run", "data": "InsertGeoIP completed"})

	// t.Test()
	// 注册路由并启动服务器
	t.register()
	mlog.Info(mlog.H{"msg": "api.Run", "data": "Server starting on " + app.Flag.App.ListenAddr})
	if err := t.router.Run(app.Flag.App.ListenAddr); err != nil {
		mlog.Error(mlog.H{"msg": "api.Run", "err": err})
	}
	mlog.Info(mlog.H{"msg": "api.Run", "data": "Run completed"})
}

func (t *mux) Test() {

	mva := getModelFields(models.GeoIPV10{})
	mlog.Info(mlog.H{"msg": "api.Test", "data": mva})
}

// 改进的泛型函数，更简洁的API
func extractFieldV2[T interface{ ~string | ~int | ~float64 }](record []string, headerMap map[string]int, fieldName string, defaultValue T) T {
	if idx, ok := headerMap[fieldName]; ok && idx < len(record) && record[idx] != "" {
		switch any(defaultValue).(type) {
		case string:
			return any(record[idx]).(T)
		case int:
			if val, err := strconv.Atoi(record[idx]); err == nil {
				return any(val).(T)
			}
		case float64:
			if val, err := strconv.ParseFloat(record[idx], 64); err == nil {
				return any(val).(T)
			}
		}
	}
	return defaultValue
}

func (m *mux) InsertGeoIP() {
	// 检查数据库连接
	if app.DB == nil {
		mlog.Error(mlog.H{"msg": "InsertGeoIP: database connection not initialized"})
		return
	}

	csvPath := os.Getenv("GOPKG_CSV_PATH")
	if csvPath == "" {
		mlog.Error(mlog.H{"msg": "InsertGeoIP: GOPKG_CSV_PATH environment variable not set"})
		return
	}

	// 检查表中是否已有数据
	var count int64
	app.DB.Model(&models.GeoIPV10{}).Count(&count)
	if count > 0 {
		mlog.Info(mlog.H{"msg": "InsertGeoIP: database already contains data", "count": count})
		// 如果需要跳过已有数据的情况，取消下面的注释
		// mlog.Info(mlog.H{"msg": "InsertGeoIP: skipping import"})
		// return
	}

	// 打开CSV文件
	file, err := os.Open(csvPath)
	if err != nil {
		mlog.Error(mlog.H{"msg": "InsertGeoIP: failed to open CSV file", "err": err})
		return
	}
	defer file.Close()

	// 计算总行数用于显示百分比
	totalLines := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		totalLines++
	}
	if err := scanner.Err(); err != nil {
		mlog.Error(mlog.H{"msg": "InsertGeoIP: failed to count total lines", "err": err})
	}

	// 重新定位文件指针到开头
	file.Seek(0, 0)

	// 配置CSV读取器
	reader := csv.NewReader(bufio.NewReader(file))
	reader.TrimLeadingSpace = true

	// 读取并解析表头
	header, err := reader.Read()
	if err != nil {
		mlog.Error(mlog.H{"msg": "InsertGeoIP: failed to read CSV header", "err": err})
		return
	}

	// 调整总行数（减去表头）
	totalLines--

	// 创建列名映射
	headerMap := make(map[string]int)
	for i, name := range header {
		headerMap[strings.ToLower(name)] = i
	}

	// 验证必要字段
	if _, ok := headerMap["cidr"]; !ok {
		mlog.Error(mlog.H{"msg": "InsertGeoIP: required field 'cidr' not found in headers"})
		return
	}

	// 设置导入参数
	recordCount := 0
	insertedCount := 0
	batchSize := 1000
	batch := make([]models.GeoIPV10, 0, batchSize)

	// 逐行处理CSV数据
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		recordCount++

		// 处理CIDR字段
		cidrIdx := headerMap["cidr"]
		if cidrIdx >= len(record) || record[cidrIdx] == "" {
			continue
		}

		cidr := record[cidrIdx]
		// 创建GeoIP对象
		geoip := models.GeoIPV10{
			Source:         "csv_import",
			Confidence:     80,
			ISP:            extractFieldV2(record, headerMap, "isp", ""),
			Cidr:           cidr,
			ESWN:           extractFieldV2(record, headerMap, "eswn", ""),
			Continent:      extractFieldV2(record, headerMap, "continent", ""),
			Country:        extractFieldV2(record, headerMap, "country", ""),
			CountryCode:    extractFieldV2(record, headerMap, "country_code", ""),
			CountryEnglish: extractFieldV2(record, headerMap, "country_english", ""),
			Province:       extractFieldV2(record, headerMap, "province", ""),
			City:           extractFieldV2(record, headerMap, "city", ""),
			District:       extractFieldV2(record, headerMap, "district", ""),
			AreaCode:       extractFieldV2(record, headerMap, "area_code", 0),
			Longitude:      extractFieldV2(record, headerMap, "longitude", 0.0),
			Latitude:       extractFieldV2(record, headerMap, "latitude", 0.0),
			ASN:            extractFieldV2(record, headerMap, "asn", 0),
			ASNOrg:         extractFieldV2(record, headerMap, "asn_org", ""),
		}

		// 处理扩展字段
		extendData := map[string]interface{}{}
		standardFields := getModelFields(models.GeoIPV10{})
		// 添加标准字段映射以便查询
		standardFieldsMap := make(map[string]bool)
		for _, field := range standardFields {
			standardFieldsMap[field] = true
		}

		for key, idx := range headerMap {
			// 排除已处理的标准字段
			if !standardFieldsMap[key] {
				if idx < len(record) && record[idx] != "" {
					extendData[key] = record[idx]
				}
			}
		}

		if len(extendData) > 0 {
			if err := geoip.SetExtendData(extendData); err != nil {
				mlog.Error(mlog.H{"msg": "InsertGeoIP: failed to set extend data", "err": err})
			}
		}

		// 添加到批处理
		batch = append(batch, geoip)

		// 当达到批处理大小时执行批量插入
		if len(batch) >= batchSize {
			// 使用OnConflict-DoNothing子句，确保冲突的记录被跳过而不是中断整个批次
			result := app.DB.Clauses(clause.OnConflict{
				DoNothing: true,
			}).CreateInBatches(batch, batchSize)

			if result.Error != nil {
				mlog.Error(mlog.H{"msg": "InsertGeoIP: batch insert error", "err": result.Error})
			}
			insertedCount += int(result.RowsAffected)

			// 清空批处理数组
			batch = make([]models.GeoIPV10, 0, batchSize)

			// 每批次完成后记录进度
			if recordCount%10000 == 0 {
				percentage := float64(recordCount) / float64(totalLines) * 100
				mlog.Info(mlog.H{"msg": "InsertGeoIP: progress", "processed": recordCount, "inserted": insertedCount, "percentage": fmt.Sprintf("%.2f%%", percentage)})
			}
		}
	}

	// 处理最后一批数据
	if len(batch) > 0 {
		result := app.DB.Clauses(clause.OnConflict{
			DoNothing: true,
		}).CreateInBatches(batch, len(batch))

		if result.Error != nil {
			mlog.Error(mlog.H{"msg": "InsertGeoIP: final batch insert error", "err": result.Error})
		}
		insertedCount += int(result.RowsAffected)
	}

	// 更新统计信息
	mlog.Info(mlog.H{"msg": "InsertGeoIP: completed", "processed": recordCount, "inserted": insertedCount, "percentage": "100.00%"})
	app.DB.Exec(fmt.Sprintf("ANALYZE %s", models.GeoIPV10{}.TableName()))
}

// LookupIP 根据IP地址查询地理位置信息
func (m *mux) LookupIP(ip string) (*models.GeoIPV10, error) {
	if app.DB == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	// 验证IP地址格式
	if ip == "" {
		return nil, fmt.Errorf("empty IP address")
	}

	var geoip models.GeoIPV10

	// 使用网络包含查询，利用GiST索引
	// 注意：这里使用PostgreSQL特有的网络地址包含操作符 >>
	result := app.DB.Where("network >> ?", ip).Order("confidence DESC").First(&geoip)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no matching network found for IP: %s", ip)
		}
		return nil, fmt.Errorf("database query error: %v", result.Error)
	}

	return &geoip, nil
}
