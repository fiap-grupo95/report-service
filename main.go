package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/fiap/secure-systems/report-service/internal/config"
	"github.com/fiap/secure-systems/report-service/internal/consumer"
	"github.com/fiap/secure-systems/report-service/internal/handler"
	"github.com/fiap/secure-systems/report-service/internal/queue"
	"github.com/fiap/secure-systems/report-service/internal/repository"
	"github.com/fiap/secure-systems/report-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed", zap.Error(err))
	}

	// ─── New Relic ────────────────────────────────────────────────────────────
	nrApp, err := newrelic.NewApplication(newrelic.ConfigFromEnvironment())
	if err != nil {
		log.Warn("new relic not configured", zap.Error(err))
		nrApp, _ = newrelic.NewApplication(newrelic.ConfigEnabled(false))
	}

	// ─── MongoDB ──────────────────────────────────────────────────────────────
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal("mongo connect failed", zap.Error(err))
	}
	if err := mongoClient.Ping(mongoCtx, nil); err != nil {
		log.Fatal("mongo ping failed", zap.Error(err))
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.MongoDB)
	reportRepo := repository.NewReportRepository(db)
	if err := reportRepo.EnsureIndexes(context.Background()); err != nil {
		log.Warn("mongo index creation failed", zap.Error(err))
	}

	// ─── RabbitMQ ─────────────────────────────────────────────────────────────
	rmq, err := queue.NewRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		log.Fatal("rabbitmq connect failed", zap.Error(err))
	}
	defer rmq.Close()

	if err := rmq.DeclareQueue(cfg.ReportQueue); err != nil {
		log.Fatal("declare report queue failed", zap.Error(err))
	}
	if err := rmq.DeclareExchange(cfg.ReportTopic); err != nil {
		log.Fatal("declare report topic failed", zap.Error(err))
	}

	deliveries, err := rmq.Consume(cfg.ReportQueue)
	if err != nil {
		log.Fatal("consume report queue failed", zap.Error(err))
	}

	// ─── Casos de Uso ─────────────────────────────────────────────────────────
	createUC := usecase.NewCreateReportUseCase(reportRepo, rmq, cfg.ReportTopic, log)
	getUC := usecase.NewGetReportUseCase(reportRepo)

	// ─── Consumer ─────────────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go consumer.NewReportQueueConsumer(createUC, nrApp, log).Run(ctx, deliveries)

	// ─── HTTP (Gin) ───────────────────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(nrgin.Middleware(nrApp))

	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	internal := r.Group("/internal")
	{
		reportH := handler.NewReportHandler(getUC, log)
		internal.GET("/reports/:reportId", reportH.GetReport)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info("report-service started", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}
	log.Info("report-service stopped")
}
