package generic

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"lightiot/pkg/generic/endpoints"
	"net/http"
)

const (
	_APIGroupPrefix = "/api"
)

type RESTProvider interface {
	NewREST(methods []string, config interface{}) (endpoints.APIGroupVersion, error)
}

type Server struct {
	Router  *gin.Engine
	Port    string
	Methods []string
}

func (s *Server) InstallAPIs(methods []string, config interface{}, restProviders ...RESTProvider) error {
	var apiGroups []*endpoints.APIGroupVersion
	for _, restBuilder := range restProviders {
		apiGroup, err := restBuilder.NewREST(methods, config)
		if err != nil {
			return err
		}
		apiGroups = append(apiGroups, &apiGroup)
	}

	return s.installAPIGroups(apiGroups...)
}

func (s *Server) installAPIGroups(apiGroups ...*endpoints.APIGroupVersion) error {
	for _, apiGroup := range apiGroups {
		if err := s.installAPIResource(_APIGroupPrefix, *apiGroup); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) installAPIResource(apiPrefix string, apiGroup endpoints.APIGroupVersion) error {
	apiGroup.Root = apiPrefix
	return apiGroup.InstallREST(s.Router)
}

func (s *Server) Start() func(context.Context) {
	s.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowHeaders:     []string{"Content-Type", "Content-Length"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowMethods:     s.Methods,
		AllowCredentials: true,
	}))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", s.Port),
		Handler: s.Router,
	}

	go func() {
		klog.Error(srv.ListenAndServe())
	}()

	return func(ctx context.Context) {
		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			klog.Error(err)
		}
	}
}
