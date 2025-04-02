package geoip

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/lwmacct/250300-go-mod-mlog/pkg/mlog"
	"github.com/oschwald/geoip2-golang"
)

// SrvMMDB 封装MaxMind数据库相关功能
type SrvMMDB struct {
	once sync.Once
	db   []*geoip2.Reader
}

// Init 初始化MaxMind数据库
func (t *SrvMMDB) Init() {
	t.once.Do(func() {
		dirList := t.findSuffixFile(os.Getenv("GOPKG_MMDB_DIR"), ".mmdb")
		for _, v := range dirList {
			mlog.Info(mlog.H{"msg": "初始化MaxMind数据库", "file": v})
			db, err := geoip2.Open(v)
			if err != nil {
				mlog.Error(mlog.H{"msg": "打开文件失败", "file": v, "err": err.Error()})
				continue
			}
			t.db = append(t.db, db)
		}
	})
}

// GetIP 从MaxMind数据库查询IP信息
func (t *SrvMMDB) GetIP(ipAddr string) (*geoip2.City, error) {
	if len(t.db) == 0 {
		mlog.Error(mlog.H{"msg": "MaxMind数据库为空"})
		return nil, fmt.Errorf("MaxMind IP数据库未初始化")
	}

	// 解析用户提供的IP地址
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return nil, fmt.Errorf("无效的IP地址: %s", ipAddr)
	}

	// 查询IP地址信息
	record, err := t.db[0].City(ip)
	if err != nil {
		mlog.Error(mlog.H{"msg": "MaxMind IP查询失败", "ip": ipAddr, "err": err.Error()})
		return nil, err
	}

	mlog.Info(mlog.H{"msg": "MaxMind IP查询成功", "ip": ipAddr})
	return record, nil
}

// 取得某目录下指定后缀的文件路径
func (t *SrvMMDB) findSuffixFile(dirName, suffix string) []string {
	return t.findSuffixFileWithDepth(dirName, suffix, 1)
}

// 递归查找指定后缀的文件，maxDepth为最大递归深度
func (t *SrvMMDB) findSuffixFileWithDepth(dirName, suffix string, depth int) []string {
	r := []string{}

	if dirName == "" {
		return r
	}

	entries, err := os.ReadDir(dirName)
	if err != nil {
		mlog.Error(mlog.H{"msg": "读取目录失败", "dir": dirName, "err": err.Error()})
		return r
	}

	for _, entry := range entries {
		fullPath := dirName + "/" + entry.Name()

		if entry.IsDir() {
			// 递归查询子目录，最大深度为3
			if depth < 3 {
				subFiles := t.findSuffixFileWithDepth(fullPath, suffix, depth+1)
				r = append(r, subFiles...)
			}
			continue
		}

		name := entry.Name()
		// 检查文件后缀
		if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
			r = append(r, fullPath)
		}
	}

	return r
}
