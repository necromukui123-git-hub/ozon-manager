package main

import (
	"fmt"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"ozon-manager/internal/config"
	"ozon-manager/internal/handler"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/repository"
	"ozon-manager/internal/service"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := repository.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// 自动迁移（表已通过SQL脚本创建，跳过）
	// if err := repository.AutoMigrate(db); err != nil {
	// 	log.Fatalf("Failed to migrate database: %v", err)
	// }

	if err := repository.EnsureOwnerColumns(db); err != nil {
		log.Fatalf("Failed to ensure owner_id columns: %v", err)
	}

	// 创建默认管理员
	if err := repository.CreateSuperAdminUser(db); err != nil {
		log.Printf("Warning: Failed to create super admin user: %v", err)
	}

	// 初始化Repository
	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	productRepo := repository.NewProductRepository(db)
	promotionRepo := repository.NewPromotionRepository(db)
	operationLogRepo := repository.NewOperationLogRepository(db)

	// 初始化Service
	authService := service.NewAuthService(userRepo, shopRepo)
	userService := service.NewUserService(userRepo, shopRepo)
	shopService := service.NewShopService(shopRepo, userRepo)
	productService := service.NewProductService(productRepo, shopRepo, promotionRepo)
	promotionService := service.NewPromotionService(productRepo, promotionRepo, shopRepo)

	// 初始化Handler
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	shopHandler := handler.NewShopHandler(shopService)
	productHandler := handler.NewProductHandler(productService, shopService)
	promotionHandler := handler.NewPromotionHandler(promotionService, shopService)
	operationLogHandler := handler.NewOperationLogHandler(operationLogRepo)

	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
	r := gin.Default()

	// 配置CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:5174", "http://localhost:3000", "https://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: true,
	}))

	// 添加安全头部
	r.Use(middleware.SecurityHeaders())

	// API路由组
	api := r.Group("/api/v1")
	{
		// 公开路由（无需认证）
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
		}

		// 需要认证的路由
		authenticated := api.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		authenticated.Use(middleware.OperationLogMiddleware(db))
		{
			// 认证相关（所有角色）
			authenticated.POST("/auth/logout", authHandler.Logout)
			authenticated.GET("/auth/me", authHandler.GetCurrentUser)
			authenticated.PUT("/auth/password", userHandler.ChangePassword)

			// 店铺查看（所有认证用户，根据角色返回不同店铺）
			authenticated.GET("/shops", shopHandler.GetShops)
			authenticated.GET("/shops/:id", shopHandler.GetShop)

			// ========== 系统管理员专用路由 ==========
			superAdmin := authenticated.Group("/admin")
			superAdmin.Use(middleware.SuperAdminOnlyMiddleware())
			{
				// 店铺管理员管理
				superAdmin.POST("/shop-admins", userHandler.CreateShopAdmin)
				superAdmin.GET("/shop-admins", userHandler.GetShopAdmins)
				superAdmin.GET("/shop-admins/:id", userHandler.GetShopAdmin)
				superAdmin.PUT("/shop-admins/:id/status", userHandler.UpdateShopAdminStatus)
				superAdmin.PUT("/shop-admins/:id/password", userHandler.ResetShopAdminPassword)
				superAdmin.DELETE("/shop-admins/:id", userHandler.DeleteShopAdmin)

				// 系统概览
				superAdmin.GET("/overview", shopHandler.GetSystemOverview)
			}

			// ========== 店铺管理员专用路由 ==========
			shopAdmin := authenticated.Group("/my")
			shopAdmin.Use(middleware.ShopAdminOnlyMiddleware())
			{
				// 店铺管理
				shopAdmin.POST("/shops", shopHandler.CreateMyShop)
				shopAdmin.GET("/shops", shopHandler.GetMyShops)
				shopAdmin.PUT("/shops/:id", shopHandler.UpdateMyShop)
				shopAdmin.DELETE("/shops/:id", shopHandler.DeleteMyShop)

				// 员工管理
				shopAdmin.POST("/staff", userHandler.CreateStaff)
				shopAdmin.GET("/staff", userHandler.GetMyStaff)
				shopAdmin.PUT("/staff/:id/status", userHandler.UpdateStaffStatus)
				shopAdmin.PUT("/staff/:id/password", userHandler.ResetStaffPassword)
				shopAdmin.PUT("/staff/:id/shops", userHandler.UpdateStaffShops)
				shopAdmin.DELETE("/staff/:id", userHandler.DeleteStaff)
			}

			// ========== 业务操作路由（shop_admin 和 staff）==========
			business := authenticated.Group("")
			business.Use(middleware.ShopAdminOrStaffMiddleware())
			{
				// 商品管理
				products := business.Group("/products")
				{
					products.GET("", productHandler.GetProducts)
					products.GET("/:id", productHandler.GetProduct)
					products.POST("/sync", productHandler.SyncProducts)
				}

				// 促销管理
				promotions := business.Group("/promotions")
				{
					// 活动管理
					promotions.GET("/actions", promotionHandler.GetActions)
					promotions.POST("/actions/manual", promotionHandler.CreateManualAction)
					promotions.DELETE("/actions/:id", promotionHandler.DeleteAction)
					promotions.PUT("/actions/:id/display-name", promotionHandler.UpdateActionDisplayName)
					promotions.PUT("/actions/sort-order", promotionHandler.UpdateActionsSortOrder)
					promotions.POST("/sync-actions", promotionHandler.SyncActions)

					// V1 接口（保持兼容）
					promotions.POST("/batch-enroll", promotionHandler.BatchEnroll)
					promotions.POST("/process-loss", promotionHandler.ProcessLoss)
					promotions.POST("/remove-reprice-promote", promotionHandler.RemoveRepricePromote)

					// V2 接口（支持选择活动）
					promotions.POST("/batch-enroll-v2", promotionHandler.BatchEnrollV2)
					promotions.POST("/process-loss-v2", promotionHandler.ProcessLossV2)
					promotions.POST("/remove-reprice-promote-v2", promotionHandler.RemoveRepricePromoteV2)
				}

				// Excel导入导出
				excel := business.Group("/excel")
				{
					excel.POST("/import-loss", promotionHandler.ImportLoss)
					excel.POST("/import-reprice", promotionHandler.ImportReprice)
					excel.GET("/export-promotable", productHandler.ExportPromotable)
					excel.GET("/template/loss", promotionHandler.DownloadLossTemplate)
				}

				// 统计
				stats := business.Group("/stats")
				{
					stats.GET("/overview", productHandler.GetStats)
				}

				// 操作日志
				business.GET("/operation-logs", operationLogHandler.GetOperationLogs)
			}
		}
	}

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	if cfg.Server.TLS.Enabled {
		log.Printf("Starting HTTPS server on %s", addr)
		log.Printf("TLS Certificate: %s", cfg.Server.TLS.CertFile)
		log.Printf("TLS Key: %s", cfg.Server.TLS.KeyFile)
		log.Printf("Default super admin account: super_admin / admin123")
		if err := r.RunTLS(addr, cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
			log.Fatalf("Failed to start HTTPS server: %v", err)
		}
	} else {
		log.Printf("⚠️  Warning: Running HTTP server (insecure)")
		log.Printf("Server starting on %s", addr)
		log.Printf("Default super admin account: super_admin / admin123")
		if err := r.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}
}
