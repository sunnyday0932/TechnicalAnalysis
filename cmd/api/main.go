package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/sunny/technical-analysis/internal/handler"
	"github.com/sunny/technical-analysis/internal/repository"
	"github.com/sunny/technical-analysis/internal/scheduler"
	"github.com/sunny/technical-analysis/internal/service"
	"github.com/sunny/technical-analysis/internal/syncer"
)

func main() {
	_ = godotenv.Load() // load .env; ignore error if not found

	ctx := context.Background()

	// --- Database ---
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected")

	// --- Repository ---
	q := repository.New(pool)

	// --- Services ---
	stockSvc := service.NewStockService(q)
	indicatorSvc := service.NewIndicatorService(stockSvc)
	syncerInst := syncer.NewSyncer(ctx, q)
	syncSvc := service.NewSyncService(q, syncerInst)

	// --- Scheduler ---
	sched := scheduler.New(syncerInst)
	sched.Start()
	defer sched.Stop()

	// --- Handlers ---
	stockH := handler.NewStockHandler(stockSvc)
	indicatorH := handler.NewIndicatorHandler(indicatorSvc)
	syncH := handler.NewSyncHandler(syncSvc)

	// --- Router ---
	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.GET("/stocks", stockH.ListStocks)
		v1.GET("/stocks/:symbol", stockH.GetStock)
		v1.GET("/stocks/:symbol/prices", stockH.GetPrices)
		v1.GET("/stocks/:symbol/indicators", indicatorH.GetIndicator)

		v1.POST("/sync", syncH.TriggerFullSync)
		v1.POST("/sync/:symbol", syncH.TriggerSymbolSync)
		v1.GET("/sync/status", syncH.GetStatus)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
