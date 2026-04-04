package product

import (
	"net/http"

	"github.com/gin-gonic/gin"
	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	"github.com/ljj/gugu-admin-api/internal/core/enum"
	"github.com/ljj/gugu-admin-api/internal/core/support/response"
)

type Controller struct {
	productService *domainproduct.Service
}

func NewController(productService *domainproduct.Service) *Controller {
	return &Controller{productService: productService}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/products", ctrl.List)
}

func (ctrl *Controller) List(c *gin.Context) {
	language := enum.NormalizeLanguage(c.DefaultQuery("language", "KO"))
	collectionSource := c.Query("collection_source")

	var (
		items []domainproduct.LocalizedProduct
		err   error
	)
	if collectionSource != "" {
		items, err = ctrl.productService.ListByCollectionSourceLocalized(c.Request.Context(), collectionSource, language)
	} else {
		items, err = ctrl.productService.ListAllLocalized(c.Request.Context(), language)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ErrorFromCode("PRODUCT_LIST_FAILED", err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"language": language,
		"items":    toProductListResponse(items),
	}))
}

type productListItemResponse struct {
	ID                string `json:"id"`
	Market            string `json:"market"`
	ExternalProductID string `json:"external_product_id"`
	OriginalURL       string `json:"original_url"`
	Title             string `json:"title"`
	MainImageURL      string `json:"main_image_url"`
	CurrentPrice      string `json:"current_price"`
	Currency          string `json:"currency"`
	ProductURL        string `json:"product_url"`
	CollectionSource  string `json:"collection_source"`
	Language          string `json:"language"`
}

func toProductListResponse(items []domainproduct.LocalizedProduct) []productListItemResponse {
	result := make([]productListItemResponse, len(items))
	for i, item := range items {
		result[i] = productListItemResponse{
			ID:                item.ID,
			Market:            string(item.Market),
			ExternalProductID: item.ExternalProductID,
			OriginalURL:       item.OriginalURL,
			Title:             item.Title,
			MainImageURL:      item.MainImageURL,
			CurrentPrice:      item.CurrentPrice,
			Currency:          item.Currency,
			ProductURL:        item.ProductURL,
			CollectionSource:  item.CollectionSource,
			Language:          item.Language,
		}
	}
	return result
}
