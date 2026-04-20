package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
	"github.com/ljj/gugu-admin-api/internal/provider/batch"
)

type Controller struct {
	skuEnricher               *batch.SKUEnricher
	skuSnapshotUpdater        *batch.SKUSnapshotUpdater
	hotProductLoader          *batch.HotProductLoader
	priceAlertEmailDispatcher *batch.PriceAlertEmailDispatcher
}

type updatePricesRequest struct {
	CollectionSource string      `json:"collection_source"`
	Market           enum.Market `json:"market"`
	ProductIDs       []string    `json:"product_ids"`
	Currencies       []string    `json:"currencies"`
	TargetGroup      string      `json:"target_group"`
	CollectedBefore  *time.Time  `json:"collected_before"`
	Force            bool        `json:"force"`
	RequestedBy      string      `json:"requested_by"`
}

type loadHotProductsRequest struct {
	CategoryIDs  []string `json:"category_ids"`
	Keywords     string   `json:"keywords"`
	Sort         string   `json:"sort"`
	MinSalePrice string   `json:"min_sale_price"`
	MaxSalePrice string   `json:"max_sale_price"`
}

func NewController(
	skuEnricher *batch.SKUEnricher,
	skuSnapshotUpdater *batch.SKUSnapshotUpdater,
	hotProductLoader *batch.HotProductLoader,
	priceAlertEmailDispatcher *batch.PriceAlertEmailDispatcher,
) *Controller {
	return &Controller{
		skuEnricher:               skuEnricher,
		skuSnapshotUpdater:        skuSnapshotUpdater,
		hotProductLoader:          hotProductLoader,
		priceAlertEmailDispatcher: priceAlertEmailDispatcher,
	}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/batch/load-hot-products", ctrl.LoadHotProducts)
	rg.POST("/batch/enrich-skus/hot-products", ctrl.EnrichHotProducts)
	rg.POST("/batch/enrich-skus/all", ctrl.EnrichAll)
	rg.POST("/batch/update-product-prices", ctrl.UpdateProductPrices)
	rg.POST("/batch/update-sku-snapshots", ctrl.UpdateSKUSnapshots)
	rg.POST("/batch/send-price-alert-emails", ctrl.SendPriceAlertEmails)
	rg.GET("/batch/status", ctrl.GetBatchStatus)
}

func (ctrl *Controller) LoadHotProducts(c *gin.Context) {
	var req loadHotProductsRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_REQUEST", err.Error()))
		return
	}

	input := batch.HotProductLoadInput{
		CategoryIDs:  req.CategoryIDs,
		Keywords:     req.Keywords,
		Sort:         req.Sort,
		MinSalePrice: req.MinSalePrice,
		MaxSalePrice: req.MaxSalePrice,
	}

	go func() {
		ctx := context.Background()
		result, err := ctrl.hotProductLoader.LoadHotProducts(ctx, input)
		if err != nil {
			log.Printf("hot product load failed: %v", err)
			return
		}
		log.Printf("hot product load completed: requested=%d hot_saved=%d product_saved=%d skipped=%d",
			result.RequestedCount, result.HotProductSaved, result.ProductSavedCount, result.SkippedCount)
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

func (ctrl *Controller) UpdateProductPrices(c *gin.Context) {
	c.JSON(http.StatusGone, response.ErrorFromCode("DEPRECATED_ENDPOINT", "update-product-prices is deprecated; use /v1/batch/update-sku-snapshots"))
}

func (ctrl *Controller) UpdateSKUSnapshots(c *gin.Context) {
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
			Currencies:       req.Currencies,
			TargetGroup:      batch.TargetGroup(strings.ToUpper(strings.TrimSpace(req.TargetGroup))),
			CollectedBefore:  req.CollectedBefore,
			Force:            req.Force,
		},
	}

	status, err := ctrl.skuSnapshotUpdater.Preview(c.Request.Context(), batchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("BATCH_PREVIEW_FAILED", err.Error()))
		return
	}

	go func(request batch.PriceUpdateRequest) {
		result, runErr := ctrl.skuSnapshotUpdater.Run(context.Background(), request)
		if runErr != nil {
			log.Printf("sku snapshot update batch failed: %v", runErr)
			return
		}
		log.Printf("sku snapshot update batch completed: %+v", result)
	}(batchReq)

	c.JSON(http.StatusAccepted, response.SuccessWithData(gin.H{
		"message": "sku snapshot update batch started",
		"job":     status,
	}))
}

func (ctrl *Controller) GetBatchStatus(c *gin.Context) {
	jobType := c.Query("job_type")
	status, ok, err := ctrl.currentStatus(jobType)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorFromCode("INVALID_JOB_TYPE", err.Error()))
		return
	}
	if !ok {
		c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
			"job_type": batch.JobTypeSKUSnapshotUpdate,
			"status":   nil,
		}))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(status))
}

func (ctrl *Controller) SendPriceAlertEmails(c *gin.Context) {
	if ctrl.priceAlertEmailDispatcher == nil {
		c.JSON(http.StatusServiceUnavailable, response.ErrorFromCode(
			"PRICE_ALERT_MAILER_NOT_CONFIGURED",
			"price alert email sender is not configured",
		))
		return
	}

	status := ctrl.priceAlertEmailDispatcher.Queue(batch.TriggerTypeManual)
	go func() {
		result, err := ctrl.priceAlertEmailDispatcher.Run(context.Background(), batch.TriggerTypeManual)
		if err != nil {
			log.Printf("price alert email batch failed: %v", err)
			return
		}
		log.Printf("price alert email batch completed: %+v", result)
	}()

	c.JSON(http.StatusAccepted, response.SuccessWithData(gin.H{
		"message": "price alert email batch started",
		"job":     status,
	}))
}

func (ctrl *Controller) currentStatus(jobType string) (batch.BatchJobStatus, bool, error) {
	switch jobType {
	case "", string(batch.JobTypeSKUSnapshotUpdate):
		status, ok := ctrl.skuSnapshotUpdater.CurrentStatus()
		return status, ok, nil
	case string(batch.JobTypePriceAlertEmailScan):
		if ctrl.priceAlertEmailDispatcher == nil {
			return batch.BatchJobStatus{}, false, nil
		}
		status, ok := ctrl.priceAlertEmailDispatcher.CurrentStatus()
		return status, ok, nil
	default:
		return batch.BatchJobStatus{}, false, fmt.Errorf("unsupported job type")
	}
}

func decodeUpdatePricesRequest(c *gin.Context) (updatePricesRequest, error) {
	var req updatePricesRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		return req, err
	}
	req.Currencies = normalizeCurrencies(req.Currencies)
	if err := validateCurrencies(req.Currencies); err != nil {
		return req, err
	}
	if !batch.IsValidTargetGroup(req.TargetGroup) {
		return req, fmt.Errorf("unsupported target_group: %s", req.TargetGroup)
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

func normalizeCurrencies(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		trimmed := strings.ToUpper(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}

func validateCurrencies(values []string) error {
	for _, value := range values {
		if !slices.Contains(enum.SupportedCurrencies, value) {
			return fmt.Errorf("unsupported currency: %s", value)
		}
	}
	return nil
}
