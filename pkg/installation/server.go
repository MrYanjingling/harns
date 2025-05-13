package installation

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic"
	v1 "lightiot/pkg/installation/v1"
	"net/http"
	"net/url"
	"os"
)

const group = "/api/installation/v1"

func InstallHandlers(router *gin.Engine, mgr *Manager) {
	v1 := router.Group(group)
	{
		v1.POST("/installations", createInstallation(mgr))
		v1.GET("/installations", listInstallations(mgr))
		v1.DELETE("/installations/:id", deleteInstallations(mgr))

		v1.PUT("/installations/:id/images", uploadImage(mgr))
		v1.GET("/installations/:id/images", listImages(mgr))
		v1.GET("/installations/:id/images/:name/content", getImageContent(mgr))
		v1.DELETE("/installations/:id/images/:name", deleteImageContent(mgr))
	}
}

func createInstallation(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		obj := &v1.Installation{}
		if err := c.ShouldBindJSON(obj); err != nil {
			klog.V(3).InfoS("Failed to parse installation", "err", err)
			c.JSON(http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		ri, err := m.CreateInstallation(obj)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewMultiError(err))
			return
		}

		// TODO use different scheme
		c.Header(apis.ETag, ri.GetVersion())
		c.Header(apis.Location, fmt.Sprintf("http://%s%s/%s", c.Request.Host, c.Request.RequestURI, ri.ID))
		c.JSON(http.StatusCreated, ri)
	}
}

func listInstallations(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Request.URL.Query()
		filter := installFilter{}
		if len(query) > 0 {
			v := query.Get(apis.Filter)
			if len(v) > 0 {
				if err := json.Unmarshal([]byte(v), &filter); err != nil {
					klog.V(3).InfoS("Failed to parse filter", "err", err)
					c.JSON(http.StatusBadRequest, response.ErrMalformedJSON)
					return
				}
			}
		}
		ris := m.listInstallations(filter)
		for _, ri := range ris {
			if ri.Type == v1.InstallationTypeUI && len(m.listImages(ri.ID)) > 0 {
				ri.Images = &Images{
					Href: fmt.Sprintf("http://%s%s/%s/images", c.Request.Host, c.Request.RequestURI, ri.ID),
				}
			}
		}

		c.JSON(http.StatusOK, &map[string]interface{}{"installations": ris})
	}
}

func deleteInstallations(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		ri, err := m.deleteInstallationById(id)
		if err != nil {
			if os.IsNotExist(err) {
				c.Status(http.StatusNotFound)
			} else if os.IsPermission(err) {
				c.Status(http.StatusForbidden)
			} else {
				klog.InfoS("Failed to delete installation", "err", err)
				c.Status(http.StatusInternalServerError)
			}
			return
		}
		c.JSON(http.StatusOK, ri)
	}
}

func uploadImage(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		name := ""
		query := c.Request.URL.Query()
		if len(query) > 0 {
			name = query.Get("name")
		}
		if len(name) == 0 {
			c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrImageNameNotFound))
			return
		}
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrImageFileNotFound))
			return
		}
		err = m.saveImage(id, name, file)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewMultiError(err))
			return
		}
		c.Status(http.StatusOK)
	}
}

func listImages(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		images := m.listImages(id)

		uri := url.URL{
			Scheme: generic.GetScheme(c.Request),
			Host:   c.Request.Host,
		}
		for _, image := range images {
			href := uri
			href.Path = fmt.Sprintf("%s/%s/content", c.Request.RequestURI, image.Name)
			image.Href = href.String()
		}

		c.JSON(http.StatusOK, &map[string]interface{}{"image": images})
	}
}

func getImageContent(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		name := c.Param("name")
		res := m.getImage(id, name)
		if len(res) > 0 {
			c.File(res)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func deleteImageContent(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		name := c.Param("name")
		err := m.deleteImage(id, name)
		if os.IsNotExist(err) {
			c.Status(http.StatusNotFound)
			return
		} else if err != nil {
			klog.V(3).InfoS("Failed to delete image", "installation", id, "name", name, "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
