package rest

import (
	"github.com/gin-gonic/gin"
	"lightiot/pkg/resources/service"
)

func Route(s *service.Service, group *gin.RouterGroup) {
	group.GET("/:resource", List(s))
	group.POST("/:resource", Create(s))
}
