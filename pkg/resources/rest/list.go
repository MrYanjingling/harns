package rest

import (
	"github.com/gin-gonic/gin"
	repo "lightiot/pkg/repository"
	"lightiot/pkg/resources"
	"net/http"
	"strconv"
	"strings"
)

func List(f repo.Finder[resources.ResourceName, repo.Record]) gin.HandlerFunc {
	return func(c *gin.Context) {
		resource := c.Param("resource")

		query := &repo.Query{
			Sort:       make(map[string]repo.SortOrder),
			Projection: make(map[string]struct{}),
			Distinct:   false,
		}

		if skipStr := c.Query("skip"); skipStr != "" {
			if skip, err := strconv.ParseUint(skipStr, 10, 64); err == nil && skip >= 0 {
				query.Skip = skip
			}
		}
		if limitStr := c.Query("limit"); limitStr != "" {
			if limit, err := strconv.ParseUint(limitStr, 10, 64); err == nil && limit > 0 {
				query.Limit = limit
			}
		}

		if sortStr := c.Query("sort"); sortStr != "" {
			pairs := strings.Split(sortStr, ",")
			for _, pair := range pairs {
				kv := strings.Split(pair, ":")
				if len(kv) == 2 {
					key := kv[0]
					order := repo.SortOrder(kv[1])
					if order == repo.SortOrderAsc || order == repo.SortOrderDesc {
						query.Sort[key] = order
					}
				}
			}
		}

		if projectionStr := c.Query("projection"); projectionStr != "" {
			fields := strings.Split(projectionStr, ",")
			for _, field := range fields {
				query.Projection[field] = struct{}{}
			}
		}

		if distinctStr := c.Query("distinct"); distinctStr != "" {
			if distinct, err := strconv.ParseBool(distinctStr); err == nil {
				query.Distinct = distinct
			}
		}

		if filterStr := c.Query("filter"); filterStr != "" {
			query.Filter = parseFilter(filterStr)
		}

		results, err := f.Find(resources.ResourceName(resource), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, results)
	}
}

func parseFilter(filterStr string) repo.Filter {
	return nil
}
