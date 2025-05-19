package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"lightiot/pkg/repository"
	"lightiot/pkg/resources/agent"
	"lightiot/pkg/resources/rest"
	"lightiot/pkg/resources/service"
	"lightiot/pkg/resources/thing"
	"lightiot/pkg/resources/thingtype"
	"net/http"
)

func main() {
	engine := gin.Default()
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowHeaders:     []string{"Content-Type", "Content-Length"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowMethods:     []string{"*"},
		AllowCredentials: true,
	}))
	group := engine.Group("/api/v1")
	repo := repository.NewSqlite(":memory:")
	srv := service.New(repo)
	srv.AddResource(thingtype.Resource())
	srv.AddResource(thing.Resource())
	srv.AddResource(agent.Resource())
	rest.Route(&srv, group)

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", 8000),
		Handler: engine,
	}

	klog.Error(server.ListenAndServe())
}
