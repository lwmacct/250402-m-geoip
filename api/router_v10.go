package api

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/lwmacct/250402-m-geoip/api/v10/geoip"
)

type routerV10 struct {
	once   sync.Once
	router *gin.RouterGroup
}

func (t *routerV10) Init(router *gin.RouterGroup) {
	t.once.Do(func() {
		t.router = router
		t.Register()
	})
}

func (t *routerV10) Register() {
	geoip.New(t.router)

}
