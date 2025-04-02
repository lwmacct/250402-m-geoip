package models

import (
	"encoding/json"
	"fmt"

	"gorm.io/datatypes"
	"gorm.io/gorm"
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
