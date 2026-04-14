package router

import (
	"github.com/gin-gonic/gin"
	ragcontroller "github.com/milabo0718/offer-pilot/backend/controller/rag"
)

func RAGRouter(r *gin.RouterGroup, rc *ragcontroller.RAGController) {
	if rc == nil {
		return
	}

	ragGroup := r.Group("/rag")
	{
		ragGroup.GET("/health", rc.Health)
		ragGroup.POST("/index/init", rc.InitIndex)
		ragGroup.POST("/ingest", rc.Ingest)
		ragGroup.POST("/search", rc.Search)
	}
}
