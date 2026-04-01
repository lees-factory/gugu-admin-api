package aliexpress

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	domaintoken "github.com/ljj/gugu-admin-api/internal/core/domain/token"
)

// PlatformProductClient implements Client interface using AliExpress Open Platform directly.
type PlatformProductClient struct {
	affiliateClient    *PlatformClient
	dropshippingClient *PlatformClient
	tokenService       TokenProvider
}

// TokenProvider retrieves access tokens from DB.
type TokenProvider interface {
	GetAccessToken(ctx context.Context, appType domaintoken.AppType) (string, error)
}

type PlatformProductConfig struct {
	AffiliateClient    *PlatformClient
	DropshippingClient *PlatformClient
	TokenService       TokenProvider
}

func NewPlatformProductClient(cfg PlatformProductConfig) *PlatformProductClient {
	return &PlatformProductClient{
		affiliateClient:    cfg.AffiliateClient,
		dropshippingClient: cfg.DropshippingClient,
		tokenService:       cfg.TokenService,
	}
}

// --- QueryHotProducts ---

type hotProductQueryAPIResponse struct {
	RespResult struct {
		Result struct {
			Products hotProductItemsContainer `json:"products"`
		} `json:"result"`
	} `json:"resp_result"`
}

type hotProductQueryTopLevelResponse struct {
	Response hotProductQueryAPIResponse `json:"aliexpress_affiliate_hotproduct_query_response"`
}

type hotProductItemsContainer struct {
	Product []hotProductItem `json:"product"`
}

type jsonString string

func (s *jsonString) UnmarshalJSON(data []byte) error {
	data = []byte(strings.TrimSpace(string(data)))
	if string(data) == "null" || len(data) == 0 {
		*s = ""
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = jsonString(str)
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		*s = jsonString(num.String())
		return nil
	}

	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*s = jsonString(strconv.FormatFloat(f, 'f', -1, 64))
		return nil
	}

	return fmt.Errorf("unsupported scalar: %s", string(data))
}

type hotProductItem struct {
	ProductID               jsonString `json:"product_id"`
	ProductTitle            string     `json:"product_title"`
	ProductMainImageURL     string     `json:"product_main_image_url"`
	ProductDetailURL        string     `json:"product_detail_url"`
	TargetSalePrice         string     `json:"target_sale_price"`
	TargetSalePriceCurrency string     `json:"target_sale_price_currency"`
	SalePrice               string     `json:"sale_price"`
	SalePriceCurrency       string     `json:"sale_price_currency"`
	PromotionLink           string     `json:"promotion_link"`
}

func (c *PlatformProductClient) QueryHotProducts(ctx context.Context, req HotProductQueryRequest) ([]HotProduct, error) {
	token, err := c.tokenService.GetAccessToken(ctx, domaintoken.AppTypeAffiliate)
	if err != nil {
		return nil, fmt.Errorf("get affiliate token: %w", err)
	}

	params := map[string]string{
		"target_currency": req.TargetCurrency,
		"target_language": req.TargetLanguage,
		"ship_to_country": req.ShipToCountry,
	}
	if req.Keywords != "" {
		params["keywords"] = req.Keywords
	}
	if len(req.CategoryIDs) > 0 {
		params["category_ids"] = strings.Join(req.CategoryIDs, ",")
	}
	if req.PageNo > 0 {
		params["page_no"] = fmt.Sprintf("%d", req.PageNo)
	}
	if req.PageSize > 0 {
		params["page_size"] = fmt.Sprintf("%d", req.PageSize)
	}
	if req.Sort != "" {
		params["sort"] = req.Sort
	}
	if req.MinSalePrice != "" {
		params["min_sale_price"] = req.MinSalePrice
	}
	if req.MaxSalePrice != "" {
		params["max_sale_price"] = req.MaxSalePrice
	}

	resp, err := c.affiliateClient.CallBusinessAPI(ctx, "aliexpress.affiliate.hotproduct.query", params, token)
	if err != nil {
		return nil, fmt.Errorf("call hot product query: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("hot product query error: code=%s message=%s", resp.Code, resp.Message)
	}

	items, err := parseHotProductItems(resp)
	if err != nil {
		return nil, err
	}

	result := make([]HotProduct, len(items))
	for i, item := range items {
		result[i] = HotProduct{
			ProductID:               string(item.ProductID),
			ProductTitle:            item.ProductTitle,
			ProductMainImageURL:     item.ProductMainImageURL,
			ProductDetailURL:        item.ProductDetailURL,
			PromotionLink:           item.PromotionLink,
			SalePrice:               item.SalePrice,
			SalePriceCurrency:       item.SalePriceCurrency,
			TargetSalePrice:         item.TargetSalePrice,
			TargetSalePriceCurrency: item.TargetSalePriceCurrency,
		}
	}

	return result, nil
}

func parseHotProductItems(resp *PlatformResponse) ([]hotProductItem, error) {
	var resultWrapped hotProductQueryAPIResponse
	if len(resp.Result) > 0 && json.Unmarshal(resp.Result, &resultWrapped) == nil && len(resultWrapped.RespResult.Result.Products.Product) > 0 {
		return resultWrapped.RespResult.Result.Products.Product, nil
	}

	var topLevel hotProductQueryTopLevelResponse
	if resp.RawBody != "" && json.Unmarshal([]byte(resp.RawBody), &topLevel) == nil {
		return topLevel.Response.RespResult.Result.Products.Product, nil
	}

	if len(resp.Result) > 0 {
		var resultOnly struct {
			CurrentPageNo int                      `json:"current_page_no"`
			Products      hotProductItemsContainer `json:"products"`
		}
		if json.Unmarshal(resp.Result, &resultOnly) == nil && len(resultOnly.Products.Product) > 0 {
			return resultOnly.Products.Product, nil
		}
	}

	return nil, fmt.Errorf("decode hot product response: unsupported structure; raw=%s", truncateForError(resp.RawBody))
}

// --- GetAffiliateProductDetails ---

type affiliateDetailAPIResponse struct {
	Result struct {
		Products []hotProductItem `json:"products"`
	} `json:"resp_result"`
}

func (c *PlatformProductClient) GetAffiliateProductDetails(ctx context.Context, req AffiliateProductDetailRequest) ([]AffiliateProductDetail, error) {
	token, err := c.tokenService.GetAccessToken(ctx, domaintoken.AppTypeAffiliate)
	if err != nil {
		return nil, fmt.Errorf("get affiliate token: %w", err)
	}

	params := map[string]string{
		"product_ids":     strings.Join(req.ProductIDs, ","),
		"target_currency": req.TargetCurrency,
		"target_language": req.TargetLanguage,
		"country":         req.Country,
	}

	resp, err := c.affiliateClient.CallBusinessAPI(ctx, "aliexpress.affiliate.productdetail.get", params, token)
	if err != nil {
		return nil, fmt.Errorf("call affiliate product detail: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("affiliate product detail error: code=%s message=%s", resp.Code, resp.Message)
	}

	var apiResp affiliateDetailAPIResponse
	if err := json.Unmarshal(resp.Result, &apiResp); err != nil {
		return nil, fmt.Errorf("decode affiliate product detail response: %w; raw=%s", err, truncateForError(string(resp.Result)))
	}

	result := make([]AffiliateProductDetail, len(apiResp.Result.Products))
	for i, item := range apiResp.Result.Products {
		result[i] = AffiliateProductDetail{
			ProductID:               string(item.ProductID),
			ProductTitle:            item.ProductTitle,
			ProductDetailURL:        item.ProductDetailURL,
			ProductMainImageURL:     item.ProductMainImageURL,
			SalePrice:               item.SalePrice,
			SalePriceCurrency:       item.SalePriceCurrency,
			TargetSalePrice:         item.TargetSalePrice,
			TargetSalePriceCurrency: item.TargetSalePriceCurrency,
		}
	}

	return result, nil
}

// --- GetDropshippingProduct ---

type dropshippingAPIResponse struct {
	Result struct {
		BaseInfo *dsBaseInfo   `json:"ae_item_base_info_dto"`
		SKUs     *dsSKUWrapper `json:"ae_item_sku_info_dtos"`
		Media    *dsMedia      `json:"ae_multimedia_info_dto"`
	} `json:"result"`
}

type dropshippingTopLevelResponse struct {
	Response dropshippingAPIResponse `json:"aliexpress_ds_product_get_response"`
}

type dsSKUWrapper struct {
	Items []dsSKU `json:"ae_item_sku_info_d_t_o"`
}

type dsBaseInfo struct {
	ProductID    string `json:"product_id"`
	Subject      string `json:"subject"`
	CurrencyCode string `json:"currency_code"`
}

type dsSKU struct {
	SKUID          string            `json:"sku_id"`
	ID             string            `json:"id"`
	SKUAttr        string            `json:"sku_attr"`
	OfferSalePrice string            `json:"offer_sale_price"`
	SKUPrice       string            `json:"sku_price"`
	CurrencyCode   string            `json:"currency_code"`
	Properties     *dsSKUPropWrapper `json:"ae_sku_property_dtos"`
}

type dsSKUPropWrapper struct {
	Items []dsSKUProp `json:"ae_sku_property_d_t_o"`
}

type dsSKUProp struct {
	SKUPropertyName  string `json:"sku_property_name"`
	SKUPropertyValue string `json:"sku_property_value"`
	SKUImage         string `json:"sku_image"`
}

type dsMedia struct {
	ImageURLs string `json:"image_urls"`
}

func (c *PlatformProductClient) GetDropshippingProduct(ctx context.Context, req DropshippingProductRequest) (*DropshippingProductDetail, error) {
	token, err := c.tokenService.GetAccessToken(ctx, domaintoken.AppTypeDropshipping)
	if err != nil {
		return nil, fmt.Errorf("get dropshipping token: %w", err)
	}

	params := map[string]string{
		"product_id":      req.ProductID,
		"ship_to_country": req.ShipToCountry,
	}
	if req.TargetCurrency != "" {
		params["target_currency"] = req.TargetCurrency
	}
	if req.TargetLanguage != "" {
		params["target_language"] = req.TargetLanguage
	}

	resp, err := c.dropshippingClient.CallBusinessAPI(ctx, "aliexpress.ds.product.get", params, token)
	if err != nil {
		return nil, fmt.Errorf("call dropshipping product: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("dropshipping product error: code=%s message=%s", resp.Code, resp.Message)
	}

	apiResp, err := parseDropshippingResponse(resp)
	if err != nil {
		return nil, err
	}

	var skus []dsSKU
	if apiResp.Result.SKUs != nil {
		skus = apiResp.Result.SKUs.Items
	}

	detail := &DropshippingProductDetail{
		SKUs: make([]DropshippingSKU, len(skus)),
	}

	if apiResp.Result.BaseInfo != nil {
		detail.ProductID = apiResp.Result.BaseInfo.ProductID
		detail.Subject = apiResp.Result.BaseInfo.Subject
		detail.CurrencyCode = apiResp.Result.BaseInfo.CurrencyCode
	}

	if apiResp.Result.Media != nil {
		detail.ImageURLs = splitImageURLs(apiResp.Result.Media.ImageURLs)
	}

	for i, sku := range skus {
		var props []dsSKUProp
		if sku.Properties != nil {
			props = sku.Properties.Items
		}

		mapped := DropshippingSKU{
			SKUID:          sku.SKUID,
			OriginSKUID:    sku.ID,
			SKUAttr:        sku.SKUAttr,
			Price:          sku.SKUPrice,
			OfferSalePrice: sku.OfferSalePrice,
			CurrencyCode:   sku.CurrencyCode,
		}
		if len(props) > 0 {
			mapped.ImageURL = strings.TrimSpace(props[0].SKUImage)
			mapped.Color = strings.TrimSpace(props[0].SKUPropertyValue)
			mapped.SKUName = strings.TrimSpace(props[0].SKUPropertyValue)
		}
		if len(props) > 1 {
			mapped.Size = strings.TrimSpace(props[1].SKUPropertyValue)
			if mapped.SKUName != "" {
				mapped.SKUName += " / "
			}
			mapped.SKUName += strings.TrimSpace(props[1].SKUPropertyValue)
		}
		if mapped.SKUName == "" {
			mapped.SKUName = strings.TrimSpace(sku.SKUAttr)
		}
		detail.SKUs[i] = mapped
	}

	return detail, nil
}

func hasDropshippingData(r *dropshippingAPIResponse) bool {
	return r.Result.BaseInfo != nil || (r.Result.SKUs != nil && len(r.Result.SKUs.Items) > 0)
}

func parseDropshippingResponse(resp *PlatformResponse) (*dropshippingAPIResponse, error) {
	// 1) resp.Result에 직접 파싱 시도
	if len(resp.Result) > 0 && string(resp.Result) != "null" {
		var apiResp dropshippingAPIResponse
		if err := json.Unmarshal(resp.Result, &apiResp); err == nil && hasDropshippingData(&apiResp) {
			return &apiResp, nil
		}
	}

	// 2) RawBody에서 top-level wrapper로 파싱 시도
	if resp.RawBody != "" {
		var topLevel dropshippingTopLevelResponse
		if err := json.Unmarshal([]byte(resp.RawBody), &topLevel); err == nil && hasDropshippingData(&topLevel.Response) {
			return &topLevel.Response, nil
		}

		// 3) error_response 확인
		var errResp struct {
			ErrorResponse struct {
				Code string `json:"code"`
				Msg  string `json:"msg"`
			} `json:"error_response"`
		}
		if json.Unmarshal([]byte(resp.RawBody), &errResp) == nil && errResp.ErrorResponse.Code != "" {
			return nil, fmt.Errorf("%s: %s", errResp.ErrorResponse.Code, errResp.ErrorResponse.Msg)
		}
	}

	return nil, fmt.Errorf("decode dropshipping product response: unsupported structure; rawBody=%s", truncateForError(resp.RawBody))
}

func splitImageURLs(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';'
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func truncateForError(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) <= 600 {
		return raw
	}
	return raw[:600] + "...(truncated)"
}
