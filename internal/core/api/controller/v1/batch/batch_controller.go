package batch

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
	"github.com/ljj/gugu-admin-api/internal/provider/batch"
)

type Controller struct {
	skuEnricher      *batch.SKUEnricher
	priceUpdater     *batch.PriceUpdater
	hotProductLoader *batch.HotProductLoader
}

type updatePricesRequest struct {
	CollectionSource string      `json:"collection_source"`
	Market           enum.Market `json:"market"`
	ProductIDs       []string    `json:"product_ids"`
	CollectedBefore  *time.Time  `json:"collected_before"`
	Force            bool        `json:"force"`
	RequestedBy      string      `json:"requested_by"`
}

type loadHotProductsRequest struct {
	CategoryIDs    []string `json:"category_ids"`
	Keywords       string   `json:"keywords"`
	PageNo         int      `json:"page_no"`
	PageSize       int      `json:"page_size"`
	MaxPages       int      `json:"max_pages"`
	Sort           string   `json:"sort"`
	MinSalePrice   string   `json:"min_sale_price"`
	MaxSalePrice   string   `json:"max_sale_price"`
	ShipToCountry  string   `json:"ship_to_country"`
	TargetCurrency string   `json:"target_currency"`
	TargetLanguage string   `json:"target_language"`
}

func NewController(skuEnricher *batch.SKUEnricher, priceUpdater *batch.PriceUpdater, hotProductLoader *batch.HotProductLoader) *Controller {
	return &Controller{
		skuEnricher:      skuEnricher,
		priceUpdater:     priceUpdater,
		hotProductLoader: hotProductLoader,
	}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/batch/load-hot-products", ctrl.LoadHotProducts)
	rg.POST("/batch/enrich-skus/hot-products", ctrl.EnrichHotProducts)
	rg.POST("/batch/enrich-skus/all", ctrl.EnrichAll)
	rg.POST("/batch/update-prices", ctrl.UpdatePrices)
	rg.GET("/batch/status", ctrl.GetBatchStatus)
}

func (ctrl *Controller) LoadHotProducts(c *gin.Context) {
	var req loadHotProductsRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REQUEST", err.Error()))
		return
	}

	input := batch.HotProductLoadInput{
		CategoryIDs:    req.CategoryIDs,
		Keywords:       req.Keywords,
		PageNo:         req.PageNo,
		PageSize:       req.PageSize,
		MaxPages:       req.MaxPages,
		Sort:           req.Sort,
		MinSalePrice:   req.MinSalePrice,
		MaxSalePrice:   req.MaxSalePrice,
		ShipToCountry:  req.ShipToCountry,
		TargetCurrency: req.TargetCurrency,
		TargetLanguage: req.TargetLanguage,
	}

	go func() {
		ctx := context.Background()
		result, err := ctrl.hotProductLoader.LoadHotProducts(ctx, input)
		if err != nil {
			log.Printf("hot product load failed: %v", err)
			return
		}
		log.Printf("hot product load completed: requested=%d hot_saved=%d product_saved=%d sku_saved=%d skipped=%d",
			result.RequestedCount, result.HotProductSaved, result.ProductSavedCount, result.SKUSavedCount, result.SkippedCount)
	}()

	c.JSON(http.StatusAccepted, response.SuccessWithData(gin.H{
		"message": "hot product load started",
	}))
}

func (ctrl *Controller) EnrichHotProducts(c *gin.Context) {
	go func() {
		ctx := context.Background()
		result, err := ctrl.skuEnricher.EnrichHotProducts(ctx)
		if err != nil {
			log.Printf("enrich hot products failed: %v", err)
			return
		}
		log.Printf("enrich hot products completed: %+v", result)
	}()

	c.JSON(http.StatusAccepted, response.SuccessWithData(gin.H{
		"message": "SKU enrichment for hot products started",
	}))
}

func (ctrl *Controller) EnrichAll(c *gin.Context) {
	go func() {
		ctx := context.Background()
		result, err := ctrl.skuEnricher.EnrichAll(ctx)
		if err != nil {
			log.Printf("enrich all failed: %v", err)
			return
		}
		log.Printf("enrich all completed: %+v", result)
	}()

	c.JSON(http.StatusAccepted, response.SuccessWithData(gin.H{
		"message": "SKU enrichment for all products started",
	}))
}

func (ctrl *Controller) UpdatePrices(c *gin.Context) {
	req, err := decodeUpdatePricesRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REQUEST", err.Error()))
		return
	}
	if req.Market != "" && !req.Market.IsSupported() {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_MARKET", "unsupported market"))
		return
	}

	batchReq := batch.PriceUpdateRequest{
		TriggerType: batch.TriggerTypeManual,
		RequestedBy: req.RequestedBy,
		Filter: batch.PriceUpdateFilter{
			CollectionSource: req.CollectionSource,
			Market:           req.Market,
			ProductIDs:       req.ProductIDs,
			CollectedBefore:  req.CollectedBefore,
			Force:            req.Force,
		},
	}

	status, err := ctrl.priceUpdater.Preview(c.Request.Context(), batchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("BATCH_PREVIEW_FAILED", err.Error()))
		return
	}

	go func(request batch.PriceUpdateRequest) {
		result, runErr := ctrl.priceUpdater.Run(context.Background(), request)
		if runErr != nil {
			log.Printf("price update batch failed: %v", runErr)
			return
		}
		log.Printf("price update batch completed: %+v", result)
	}(batchReq)

	c.JSON(http.StatusAccepted, response.SuccessWithData(gin.H{
		"message": "price update batch started",
		"job":     status,
	}))
}

func (ctrl *Controller) GetBatchStatus(c *gin.Context) {
	jobType := c.Query("job_type")
	if jobType != "" && jobType != string(batch.JobTypePriceUpdate) {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_JOB_TYPE", "unsupported job type"))
		return
	}

	status, ok := ctrl.priceUpdater.CurrentStatus()
	if !ok {
		c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
			"job_type": batch.JobTypePriceUpdate,
			"status":   nil,
		}))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(status))
}

func decodeUpdatePricesRequest(c *gin.Context) (updatePricesRequest, error) {
	var req updatePricesRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		return req, err
	}
	return req, nil
}

func decodeOptionalJSON(c *gin.Context, target any) error {
	if c.Request.Body == nil {
		return nil
	}

	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(target); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	return nil
}
