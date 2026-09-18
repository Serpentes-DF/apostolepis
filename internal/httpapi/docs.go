package http

import (
	"net/http"

	"github.com/Serpentes-DF/apostolepis/docs"
	"github.com/gin-gonic/gin"
	"github.com/swaggest/swgui/v5emb"
)

var swaggerUI = v5emb.New("Apostolepis API", "/docs/openapi.yaml", "/docs/")

func serveDocs(c *gin.Context) {
	if c.Param("path") == "/openapi.yaml" {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.OpenAPI)
		return
	}

	if c.Request.URL.Path == "/docs" {
		c.Request.URL.Path = "/docs/"
	}
	swaggerUI.ServeHTTP(c.Writer, c.Request)
}
