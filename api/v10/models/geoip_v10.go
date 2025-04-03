package models

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/lwmacct/250300-go-mod-mlog/pkg/mlog"
	"github.com/lwmacct/250402-m-geoip/app"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GeoIPV10 IP地理位置信息，存储网络地址段的地理位置数据
type GeoIPV10 struct {
	gorm.Model     `json:"-"`     // 内嵌 gorm.Model，包含 ID、CreatedAt、UpdatedAt 和 DeletedAt 字段
	Extend         datatypes.JSON `json:"extend" gorm:"type:jsonb;column:extend;comment:额外的扩展信息，使用 JSON 格式存储"`
	Source         string         `json:"source" gorm:"type:varchar(32);column:source;comment:数据来源（如 maxmind、ipip 等）"`
	Confidence     int            `json:"confidence" gorm:"type:smallint;column:confidence;comment:数据可信度(0-100)"`
	ISP            string         `json:"isp" gorm:"type:varchar(255);column:isp;comment:互联网服务提供商"`
	Cidr           string         `json:"cidr" gorm:"type:cidr;not null;column:cidr;comment:CIDR 网络地址段"`
	ESWN           string         `json:"eswn" gorm:"type:varchar(32);column:eswn;comment:东南西北"`
	Continent      string         `json:"continent" gorm:"type:varchar(255);column:continent;comment:大洲名称"`
	Country        string         `json:"country" gorm:"type:varchar(255);column:country;comment:国家名称"`
	CountryCode    string         `json:"country_code" gorm:"type:varchar(255);column:country_code;comment:国家代码"`
	CountryEnglish string         `json:"country_english" gorm:"type:varchar(255);column:country_english;comment:国家英文名称"`
	Province       string         `json:"province" gorm:"type:varchar(255);column:province;comment:省份/州名称"`
	City           string         `json:"city" gorm:"type:varchar(255);column:city;comment:城市名称"`
	District       string         `json:"district" gorm:"type:varchar(255);column:district;comment:区/县名称"`
	AreaCode       int            `json:"area_code" gorm:"type:int;column:area_code;comment:区域代码"`
	Latitude       float64        `json:"latitude" gorm:"type:double precision;column:latitude;comment:纬度坐标"`
	Longitude      float64        `json:"longitude" gorm:"type:double precision;column:longitude;comment:经度坐标"`
	ASN            int            `json:"asn" gorm:"type:int;column:asn;comment:自治系统编号"`
	ASNOrg         string         `json:"asn_org" gorm:"type:varchar(255);column:asn_org;comment:自治系统组织"`
}

// TableName 指定表名
func (GeoIPV10) TableName() string {
	return "geoip_v10"
}

// SetExtendData 设置额外数据到Extend字段
func (g *GeoIPV10) SetExtendData(data map[string]interface{}) error {
	if len(data) > 0 {
		extendJSON, err := json.Marshal(data)
		if err != nil {
			return err
		}
		g.Extend = extendJSON
	}
	return nil
}

// GetExtendData 从Extend字段获取额外数据
func (g *GeoIPV10) GetExtendData() (map[string]interface{}, error) {
	var data map[string]interface{}
	if len(g.Extend) > 0 {
		if err := json.Unmarshal(g.Extend, &data); err != nil {
			return nil, err
		}
	} else {
		data = make(map[string]interface{})
	}
	return data, nil
}

func (t GeoIPV10) InsertSampleData(db *gorm.DB) {
	csvPath := os.Getenv("GOPKG_CSV_PATH_250402")
	// 检查数据库连接
	if db == nil {
		mlog.Error(mlog.H{"msg": "InsertGeoIP: database connection not initialized"})
		return
	}

	if csvPath == "" {
		mlog.Error(mlog.H{"msg": "InsertGeoIP: GOPKG_CSV_PATH environment variable not set"})
		return
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
	batch := make([]GeoIPV10, 0, batchSize)

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
		geoip := GeoIPV10{
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
		standardFields := getModelFields(GeoIPV10{})
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
			batch = make([]GeoIPV10, 0, batchSize)

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
	app.DB.Exec(fmt.Sprintf("ANALYZE %s", GeoIPV10{}.TableName()))
}

// TableIndex 定义并创建表索引
// 接收数据库连接，直接执行索引创建操作
func (GeoIPV10) TableIndex(db *gorm.DB) error {
	var sourceNetworkIndexCreated bool = false

	var name string = GeoIPV10{}.TableName()
	var idxPrefix string = fmt.Sprintf("idx_%s_", name)

	// 创建GiST索引用于CIDR查询，支持IP地址范围查询
	sql1 := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING gist (cidr inet_ops)", fmt.Sprintf("%s_network", idxPrefix), name)
	if err := db.Exec(sql1).Error; err != nil {
		// 记录错误但继续执行，因为唯一性索引更重要
		db.Logger.Error(db.Statement.Context, "Failed to create network GiST index: %v", err)
	} else {
		db.Logger.Info(db.Statement.Context, "Network GiST index created successfully")
	}

	// 创建Source和Network的联合唯一索引
	sql2 := fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (source, cidr)", fmt.Sprintf("%s_source_network", idxPrefix), name)
	if err := db.Exec(sql2).Error; err != nil {
		db.Logger.Error(db.Statement.Context, "Failed to create source+network unique index: %v", err)
		return err
	} else {
		db.Logger.Info(db.Statement.Context, "Source+network unique index created successfully")
		sourceNetworkIndexCreated = true
	}

	// 如果没有成功创建唯一索引，返回错误
	if !sourceNetworkIndexCreated {
		return fmt.Errorf("failed to create unique index, data uniqueness may not be guaranteed")
	}

	return nil
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

// getModelFields 从模型结构体中提取字段名称
func getModelFields(model interface{}) []string {
	fields := []string{}

	// 获取结构体类型
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 遍历所有字段
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 处理嵌入式字段（如gorm.Model）
		if field.Anonymous {
			// 如果字段类型是结构体
			if field.Type.Kind() == reflect.Struct {
				// 递归获取嵌入式结构体的字段
				embeddedFields := getModelFields(reflect.New(field.Type).Elem().Interface())
				fields = append(fields, embeddedFields...)
			}
			continue
		}

		// 获取标签中的列名
		tagValue := field.Tag.Get("gorm")
		columnName := ""
		for _, tag := range strings.Split(tagValue, ";") {
			if strings.HasPrefix(tag, "column:") {
				columnName = strings.TrimPrefix(tag, "column:")
				break
			}
		}

		// 如果没有指定列名，使用字段名小写
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}

		fields = append(fields, columnName)
	}

	return fields
}
