package geoip

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lwmacct/250300-go-mod-mgin/pkg/mgin"
	"github.com/lwmacct/250402-m-geoip/api/v10/models"
)

// IPQueryResult 查询结果的通用结构，包含查询的IP和结果
type IPQueryResult struct {
	models.GeoIPV10
	Ip string `json:"ip"`
}

type main struct {
	mgin.Handler
	srv1 *SrvDBQuery
}

func (t *main) Register(r *gin.RouterGroup) {
	rg := r.Group("geoip")
	rg.GET(":ip", t.Get)
	rg.POST("", t.Post)
	rg.PUT("", t.Put)
	rg.DELETE("", t.Delete)
}

func (t *main) Get(c *gin.Context) {
	// 从路径参数中获取IP地址或CIDR
	input := c.Param("ip")

	// 准备返回结果数组
	var results []IPQueryResult

	// 检查是否提供了多个IP (以逗号分隔)
	if input != "" && strings.Contains(input, ",") {
		// 批量查询处理
		ips := strings.Split(input, ",")

		// 处理每个IP
		for _, ip := range ips {
			ip = strings.TrimSpace(ip) // 移除可能的空格
			if ip == "" {
				continue
			}

			result := IPQueryResult{Ip: ip}
			// result := IPQueryResult[models.GeoIPV10]{IP: ip}
			ipData, err := t.srv1.GetIPInfo(ip)

			if err == nil {
				result.GeoIPV10 = ipData
			}

			results = append(results, result)
		}
	} else {
		// 单个IP查询处理
		if input == "" {
			input = c.ClientIP() // 如果没有提供输入，使用客户端IP
		}

		// result := IPQueryResult[models.GeoIPV10]{IP: input}
		result := IPQueryResult{Ip: input}
		ipData, err := t.srv1.GetIPInfo(input)

		if err == nil {
			result.GeoIPV10 = ipData
		}

		results = append(results, result)
	}

	// 返回结果数组
	response := mgin.Response[[]IPQueryResult]{
		Code: http.StatusOK,
		Msg:  "success",
		Data: results,
	}
	c.JSON(response.Code, response)
}

// Post 处理批量IP查询请求
func (t *main) Post(c *gin.Context) {
	// 直接使用IP地址数组作为请求
	var ips []string

	// 绑定请求数据
	if err := c.ShouldBindJSON(&ips); err != nil {
		response := mgin.Response[string]{
			Code: http.StatusBadRequest,
			Msg:  "无效的请求数据: " + err.Error(),
			Data: "",
		}
		c.JSON(response.Code, response)
		return
	}

	// 检查IP列表是否为空
	if len(ips) == 0 {
		response := mgin.Response[string]{
			Code: http.StatusBadRequest,
			Msg:  "IP列表不能为空",
			Data: "",
		}
		c.JSON(response.Code, response)
		return
	}

	// 存储查询结果
	var results []IPQueryResult

	// 验证并处理每个IP
	for _, ip := range ips {
		// 移除空格
		ip = strings.TrimSpace(ip)

		// 验证IP非空
		if ip == "" {
			continue
		}

		// 基本格式验证 - 简单检查是否包含字母数字和常见IP符号
		if !isValidIPFormat(ip) {
			result := IPQueryResult{Ip: ip}
			results = append(results, result)
			continue
		}

		result := IPQueryResult{Ip: ip}
		ipData, err := t.srv1.GetIPInfo(ip)

		if err == nil {
			result.GeoIPV10 = ipData
		}

		results = append(results, result)
	}

	// 返回查询结果
	response := mgin.Response[[]IPQueryResult]{
		Code: http.StatusOK,
		Msg:  "success",
		Data: results,
	}
	c.JSON(response.Code, response)
}

// isValidIPFormat 简单验证IP格式是否合法
func isValidIPFormat(ip string) bool {
	// 验证IP字符串只包含合法的IP字符：数字、点、冒号（IPv6）、斜杠（CIDR表示法）
	for _, c := range ip {
		if !((c >= '0' && c <= '9') || c == '.' || c == ':' || c == '/' || c == '-') {
			return false
		}
	}
	return true
}

func New(router *gin.RouterGroup) *main {
	t := &main{}
	t.Register(router)
	// 确保服务已初始化
	t.srv1 = new(SrvDBQuery).Init()
	return t
}
