package rag

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/milabo0718/offer-pilot/backend/common/code"
	"github.com/milabo0718/offer-pilot/backend/controller"
	"github.com/milabo0718/offer-pilot/backend/model"
	ragservice "github.com/milabo0718/offer-pilot/backend/service/rag"
)

type (
	RAGInitIndexResponse struct {
		controller.Response
	}

	RAGHealthResponse struct {
		controller.Response
		Health model.RAGHealthStatus `json:"health,omitempty"`
	}

	RAGIngestResponse struct {
		controller.Response
		Stats model.IngestStats `json:"stats,omitempty"`
	}

	RAGSearchResponse struct {
		controller.Response
		Results []model.SearchResult `json:"results,omitempty"`
	}
)

// RAGController 提供最小可用的索引初始化、入库和检索接口。
type RAGController struct {
	ingestService *ragservice.IngestService
	searchService *ragservice.SearchService
	defaultDir    string
}

func NewRAGController(ingestService *ragservice.IngestService, searchService *ragservice.SearchService, defaultDir string) *RAGController {
	return &RAGController{
		ingestService: ingestService,
		searchService: searchService,
		defaultDir:    defaultDir,
	}
}

func (rc *RAGController) InitIndex(ctx *gin.Context) {
	res := new(RAGInitIndexResponse)
	if err := rc.ingestService.EnsureIndex(ctx); err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.RAGIndexInitFail))
		return
	}
	res.Success()
	ctx.JSON(http.StatusOK, res)
}

func (rc *RAGController) Health(ctx *gin.Context) {
	res := new(RAGHealthResponse)
	health := rc.ingestService.Health(ctx)

	if !health.RedisReachable || !health.RedisSearchReady {
		res.CodeOf(code.RAGIndexInitFail)
		res.Health = health
		ctx.JSON(http.StatusOK, res)
		return
	}

	res.Success()
	res.Health = health
	ctx.JSON(http.StatusOK, res)
}

func (rc *RAGController) Ingest(ctx *gin.Context) {
	req := new(model.RAGIngestRequest)
	res := new(RAGIngestResponse)

	// 允许空请求体，这样前端或脚本可以直接触发默认目录入库。
	if ctx.Request.ContentLength > 0 {
		if err := ctx.ShouldBindJSON(req); err != nil {
			ctx.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
			return
		}
	}

	dir := strings.TrimSpace(req.Directory)
	if dir == "" {
		dir = rc.defaultDir
	}
	if dir == "" {
		ctx.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	stats, err := rc.ingestService.IngestDirectory(ctx, dir)
	if err != nil {
		res.CodeOf(code.RAGIngestFail)
		res.Stats = stats
		ctx.JSON(http.StatusOK, res)
		return
	}

	res.Success()
	res.Stats = stats
	ctx.JSON(http.StatusOK, res)
}

func (rc *RAGController) Search(ctx *gin.Context) {
	req := new(model.RAGSearchRequest)
	res := new(RAGSearchResponse)

	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	results, err := rc.searchService.SearchRelevantChunks(ctx, req.Query, req.TopK, &req.Filter)
	if err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.RAGSearchFail))
		return
	}

	res.Success()
	res.Results = results
	ctx.JSON(http.StatusOK, res)
}
