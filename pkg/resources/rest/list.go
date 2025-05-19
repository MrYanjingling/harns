package rest

import (
	"github.com/gin-gonic/gin"
	repo "lightiot/pkg/repository"
	"lightiot/pkg/resources"
	"net/http"
)

func List(f repo.Finder[resources.ResourceName, repo.Record]) gin.HandlerFunc {
	return func(c *gin.Context) {
		resource := c.Param("resource")
		// TODO: parse query
		results, err := f.Find(resources.ResourceName(resource), &repo.Query{})
		if err != nil {
			return
		}
		c.JSON(http.StatusOK, results)
	}
}
