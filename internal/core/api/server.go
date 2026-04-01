package api

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ljj/gugu-admin-api/internal/clients/aliexpress"
	batchctrl "github.com/ljj/gugu-admin-api/internal/core/api/controller/v1/batch"
	tokenctrl "github.com/ljj/gugu-admin-api/internal/core/api/controller/v1/token"
	userctrl "github.com/ljj/gugu-admin-api/internal/core/api/controller/v1/user"
	"github.com/ljj/gugu-admin-api/internal/core/api/middleware"
	domainproduct "github.com/ljj/gugu-admin-api/internal/core/domain/product"
	domaintoken "github.com/ljj/gugu-admin-api/internal/core/domain/token"
	domainuser "github.com/ljj/gugu-admin-api/internal/core/domain/user"
	"github.com/ljj/gugu-admin-api/internal/provider/batch"
	dbcorehotproduct "github.com/ljj/gugu-admin-api/internal/storage/dbcore/hotproduct"
	dbcorepricehistory "github.com/ljj/gugu-admin-api/internal/storage/dbcore/pricehistory"
	dbcoreproduct "github.com/ljj/gugu-admin-api/internal/storage/dbcore/product"
	dbcoretoken "github.com/ljj/gugu-admin-api/internal/storage/dbcore/token"
	dbcoreuser "github.com/ljj/gugu-admin-api/internal/storage/dbcore/user"
	"github.com/ljj/gugu-admin-api/internal/support/clock"
	"github.com/ljj/gugu-admin-api/internal/support/config"
	"github.com/ljj/gugu-admin-api/internal/support/id"
)

func NewServer(cfg config.Config, db *sql.DB) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/openapi.yml", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		path := findOpenAPIFile()
		if path == "" {
			c.String(http.StatusNotFound, "openapi.yml not found")
			return
		}
		c.File(path)
	})

	v1 := r.Group("/v1")
	v1.Use(middleware.AdminAuth())
	{
		registerRoutes(v1, cfg, db)
	}

	return r
}

func registerRoutes(rg *gin.RouterGroup, cfg config.Config, db *sql.DB) {
	// Infra
	productRepo := dbcoreproduct.NewSQLCRepository(db)
	skuRepo := dbcoreproduct.NewSKUSQLCRepository(db)
	userRepo := dbcoreuser.NewSQLCRepository(db)
	tokenRepo := dbcoretoken.NewSQLCRepository(db)
	hotProductRepo := dbcorehotproduct.NewSQLCRepository(db)
	priceHistoryRepo := dbcorepricehistory.NewRepository(db)
	idGen := id.NewGenerator()
	clk := clock.New()

	// Domain
	productFinder := domainproduct.NewFinder(productRepo)
	productWriter := domainproduct.NewWriter(productRepo)
	productService := domainproduct.NewService(productFinder, productWriter, skuRepo, idGen, clk)
	userFinder := domainuser.NewFinder(userRepo)
	userService := domainuser.NewService(userFinder)
	tokenService := domaintoken.NewService(tokenRepo, idGen, clk)

	// Clients
	affiliatePlatformClient := aliexpress.NewPlatformClient(aliexpress.PlatformConfig{
		AppKey:    cfg.AliExpressAppKey,
		AppSecret: cfg.AliExpressAppSecret,
	})
	dropshippingPlatformClient := aliexpress.NewPlatformClient(aliexpress.PlatformConfig{
		AppKey:    cfg.AliExpressDSAppKey,
		AppSecret: cfg.AliExpressDSAppSecret,
	})
	aliexpressClient := aliexpress.NewPlatformProductClient(aliexpress.PlatformProductConfig{
		AffiliateClient:    affiliatePlatformClient,
		DropshippingClient: dropshippingPlatformClient,
		TokenService:       tokenService,
	})

	// Batch
	batchStatusStore := batch.NewBatchStatusStore()
	skuEnricher := batch.NewSKUEnricher(
		productService,
		aliexpressClient,
		priceHistoryRepo,
		45*time.Second,
		75*time.Second,
	)
	priceSource := batch.NewAliExpressPriceSource(aliexpressClient, 500*time.Millisecond)
	priceUpdater := batch.NewPriceUpdater(
		productService,
		batchStatusStore,
		priceSource,
		priceHistoryRepo,
	)
	hotProductLoader := batch.NewHotProductLoader(aliexpressClient, productService, hotProductRepo, priceHistoryRepo, idGen)
	if cfg.PriceUpdateScheduleEnabled {
		priceUpdateScheduler := batch.NewPriceUpdateScheduler(
			priceUpdater,
			cfg.PriceUpdateScheduleInterval,
		)
		priceUpdateScheduler.Start(context.Background())
	}
	if cfg.TokenRefreshEnabled {
		tokenRefreshScheduler := batch.NewTokenRefreshScheduler(
			tokenService,
			affiliatePlatformClient,
			cfg.TokenRefreshInterval,
		)
		tokenRefreshScheduler.Start(context.Background())
	}

	// Controllers
	batchController := batchctrl.NewController(skuEnricher, priceUpdater, hotProductLoader)
	batchController.RegisterRoutes(rg)
	tokenController := tokenctrl.NewController(tokenService, affiliatePlatformClient, dropshippingPlatformClient)
	tokenController.RegisterRoutes(rg)
	userController := userctrl.NewController(userService)
	userController.RegisterRoutes(rg)
}

func findOpenAPIFile() string {
	candidates := []string{
		"openapi.yml",
		"/home/ubuntu/gugu-admin-api/openapi.yml",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
