// @title Lab 2-4 REST API
// @version 1.0
// @description REST API для управления категориями и продуктами с JWT и OAuth2 авторизацией.
// @host localhost:4200
// @BasePath /
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name access_token
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/lab2/rest-api/internal/config"
	_ "github.com/lab2/rest-api/docs"

	//"github.com/lab2/rest-api/internal/domain"
	categoryhandler "github.com/lab2/rest-api/internal/category/handler"
	categoryrepo "github.com/lab2/rest-api/internal/category/repository"
	categorysvc "github.com/lab2/rest-api/internal/category/service"
	"github.com/lab2/rest-api/internal/middleware"
	producthandler "github.com/lab2/rest-api/internal/product/handler"
	productrepo "github.com/lab2/rest-api/internal/product/repository"
	productsvc "github.com/lab2/rest-api/internal/product/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// Библиотека миграций
	// Сам пакет migrate
	"github.com/golang-migrate/migrate/v4"
	// Драйверы импортируем с подчеркиванием _, чтобы они не конфликтовали с gorm.io/driver/postgres
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// Auth модуль (с алиасами!)
	authhandler "github.com/lab2/rest-api/internal/auth/handler"
	authmiddleware "github.com/lab2/rest-api/internal/auth/middleware"
	authrepo "github.com/lab2/rest-api/internal/auth/repository"
	authservice "github.com/lab2/rest-api/internal/auth/service"
)

// runMigrations запускает миграции базы данных
func runMigrations(dsn string) error {
	m, err := migrate.New(
		"file:///app/internal/migrations", // абсолютный путь
		dsn,
	)
	if err != nil {
		return fmt.Errorf("ошибка инициализации миграций: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("ошибка применения миграций: %w", err)
	}

	return nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	// Запускаем миграции перед стартом приложения
	if err := runMigrations(cfg.MigrationDSN()); err != nil {
		log.Fatalf("Ошибка миграций: %v", err)
	}
	log.Println("Миграции успешно применены")

	// ========== ИНИЦИАЛИЗАЦИЯ AUTH МОДУЛЯ ==========

	// Репозитории
	userRepo := authrepo.NewUserRepository(db)
	tokenRepo := authrepo.NewTokenRepository(db)

	// Сервисы
	passwordService := authservice.NewPasswordService()

	// Парсим время жизни токенов из конфига
	accessDur, _ := time.ParseDuration(cfg.JWTAccessExpiration)
	refreshDur, _ := time.ParseDuration(cfg.JWTRefreshExpiration)

	jwtService := authservice.NewJWTService(
		cfg.JWTAccessSecret,
		cfg.JWTRefreshSecret,
		accessDur,
		refreshDur,
	)
	// Репозиторий для токенов сброса пароля
	passwordResetRepo := authrepo.NewPasswordResetRepository(db)
	authService := authservice.NewAuthService(
		userRepo,
		tokenRepo,
		passwordService,
		jwtService,
		passwordResetRepo,
	)

	// Хендлер
	authHandler := authhandler.NewAuthHandler(authService)
	passwordHandler := authhandler.NewPasswordHandler(authService)

	// ========== OAUTH ==========

	// Конфигурация OAuth
	oauthConfig := &authservice.OAuthConfig{
		YandexClientID:     cfg.YandexClientID,
		YandexClientSecret: cfg.YandexClientSecret,
		YandexRedirectURI:  cfg.YandexCallbackURL,
	}
	// Сервис
	oauthService := authservice.NewOAuthService(
		userRepo,
		tokenRepo,
		passwordService,
		jwtService,
		oauthConfig,
	)

	// Хендлер
	oauthHandler := authhandler.NewOAuthHandler(oauthService)

	// Middleware
	authMW := authmiddleware.AuthMiddleware(jwtService, tokenRepo)
	/*
		if err := db.AutoMigrate(&domain.Category{}, &domain.Product{}); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	*/
	categoryRepo := categoryrepo.NewCategoryRepository(db)
	productRepo := productrepo.NewProductRepository(db)
	categorySvc := categorysvc.NewCategoryService(categoryRepo, productRepo)
	productSvc := productsvc.NewProductService(productRepo, categoryRepo)
	categoryHandler := categoryhandler.NewCategoryHandler(categorySvc)
	productHandler := producthandler.NewProductHandler(productSvc)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.Recovery())
	if cfg.AppEnv != "production" {
		r.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	// ========== PUBLIC ROUTES (без авторизации) ==========
	publicAuth := r.Group("/auth")
	{
		publicAuth.POST("/register", authHandler.Register)
		publicAuth.POST("/login", authHandler.Login)
		publicAuth.POST("/refresh", authHandler.Refresh)

		publicAuth.POST("/forgot-password", passwordHandler.ForgotPassword)
		publicAuth.POST("/reset-password", passwordHandler.ResetPassword)
		// OAUTH ROUTES
		publicAuth.GET("/oauth/:provider", oauthHandler.InitOAuth)
		publicAuth.GET("/oauth/:provider/callback", oauthHandler.OAuthCallback)
	}

	// ========== PROTECTED ROUTES (с авторизацией) ==========
	protectedAuth := r.Group("/auth")
	protectedAuth.Use(authMW) // ← применяем middleware
	{
		protectedAuth.GET("/whoami", authHandler.WhoAmI)
		protectedAuth.POST("/logout", authHandler.Logout)
		protectedAuth.POST("/logout-all", authHandler.LogoutAll)
	}

	// Categories (с защитой)
	categories := r.Group("/categories")
	categories.Use(authMW)
	{
		categories.GET("", categoryHandler.List)
		categories.GET("/:id", categoryHandler.GetByID)
		categories.POST("", categoryHandler.Create)
		categories.PUT("/:id", categoryHandler.Update)
		categories.PATCH("/:id", categoryHandler.Patch)
		categories.DELETE("/:id", categoryHandler.Delete)
	}

	// Products (с защитой)
	products := r.Group("/products")
	products.Use(authMW)
	{
		products.GET("", productHandler.List)
		products.GET("/:id", productHandler.GetByID)
		products.POST("", productHandler.Create)
		products.PUT("/:id", productHandler.Update)
		products.PATCH("/:id", productHandler.Patch)
		products.DELETE("/:id", productHandler.Delete)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
