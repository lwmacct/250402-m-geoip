package geoip

import (
	"fmt"
	"net"
	"sync"

	"github.com/lwmacct/250300-go-mod-mlog/pkg/mlog"
	"github.com/lwmacct/250402-m-geoip/api/v10/models"
	"github.com/lwmacct/250402-m-geoip/app"
)

// SrvDBQuery 主服务结构体
type SrvDBQuery struct {
	once sync.Once
}

// Init 初始化服务
func (t *SrvDBQuery) Init() *SrvDBQuery {
	t.once.Do(func() {
	})
	return t
}

// GetIPInfo 根据输入自动区分IP和CIDR进行查询
func (t *SrvDBQuery) GetIPInfo(input string) (models.GeoIPV10, error) {
	// 检查数据库连接
	if app.DB == nil {
		mlog.Error(mlog.H{"msg": "数据库连接未初始化"})
		return models.GeoIPV10{}, fmt.Errorf("数据库连接未初始化")
	}

	// 判断输入是IP还是CIDR
	if _, _, err := net.ParseCIDR(input); err == nil {
		// 输入是CIDR格式
		return t.queryByCIDR(input)
	} else {
		// 输入可能是IP格式
		ip := net.ParseIP(input)
		if ip == nil {
			return models.GeoIPV10{}, fmt.Errorf("无效的输入格式: %s", input)
		}
		return t.queryByIP(ip.String())
	}
}

// queryByIP 通过IP地址查询信息
func (t *SrvDBQuery) queryByIP(ipAddr string) (models.GeoIPV10, error) {
	// 验证IP地址格式
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return models.GeoIPV10{}, fmt.Errorf("无效的IP地址: %s", ipAddr)
	}

	// 使用PostgreSQL的网络包含查询操作符 >>
	var geoip models.GeoIPV10
	result := app.DB.Where("cidr >> ?", ipAddr).Order("confidence DESC").First(&geoip)

	if result.Error != nil {
		mlog.Error(mlog.H{"msg": "IP查询失败", "ip": ipAddr, "err": result.Error.Error()})
		return models.GeoIPV10{}, fmt.Errorf("数据库查询错误: %v", result.Error)
	}

	mlog.Info(mlog.H{"msg": "数据库IP查询成功", "ip": ipAddr})
	return geoip, nil
}

// queryByCIDR 通过CIDR查询信息
func (t *SrvDBQuery) queryByCIDR(cidr string) (models.GeoIPV10, error) {
	// 验证CIDR格式
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return models.GeoIPV10{}, fmt.Errorf("无效的CIDR格式: %s, %v", cidr, err)
	}

	// 直接匹配CIDR
	var geoip models.GeoIPV10
	result := app.DB.Where("cidr = ?", cidr).Order("confidence DESC").First(&geoip)

	if result.Error != nil {
		mlog.Error(mlog.H{"msg": "CIDR查询失败", "cidr": cidr, "err": result.Error.Error()})
		return models.GeoIPV10{}, fmt.Errorf("数据库查询错误: %v", result.Error)
	}

	mlog.Info(mlog.H{"msg": "数据库CIDR查询成功", "cidr": cidr})
	return geoip, nil
}
