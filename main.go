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
	"github.com/fiap/secure-systems/report-service/internal/logging"
	"github.com/fiap/secure-systems/report-service/internal/queue"
	"github.com/fiap/secure-systems/report-service/internal/repository"
	"github.com/fiap/secure-systems/report-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("config load failed: " + err.Error())
	}

	// ─── New Relic ────────────────────────────────────────────────────────────
	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	if err != nil {
		nrApp, _ = newrelic.NewApplication(newrelic.ConfigEnabled(false))
	}

	// ─── Logging (deve ser inicializado após o New Relic) ─────────────────────
	logging.Init(nrApp)
	log := logging.Logger()

	// ─── MongoDB ──────────────────────────────────────────────────────────────
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal().Err(err).Msg("mongo connect failed")
	}
	if err := mongoClient.Ping(mongoCtx, nil); err != nil {
		log.Fatal().Err(err).Msg("mongo ping failed")
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.MongoDB)
	reportRepo := repository.NewReportRepository(db)
	if err := reportRepo.EnsureIndexes(context.Background()); err != nil {
		log.Warn().Err(err).Msg("mongo index creation failed")
	}

	// ─── RabbitMQ ─────────────────────────────────────────────────────────────
	rmq, err := queue.NewRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		log.Fatal().Err(err).Msg("rabbitmq connect failed")
	}
	defer rmq.Close()

	if err := rmq.DeclareQueue(cfg.ReportQueue); err != nil {
		log.Fatal().Err(err).Msg("declare report queue failed")
	}
	if err := rmq.DeclareExchange(cfg.ReportTopic); err != nil {
		log.Fatal().Err(err).Msg("declare report topic failed")
	}

	deliveries, err := rmq.Consume(cfg.ReportQueue)
	if err != nil {
		log.Fatal().Err(err).Msg("consume report queue failed")
	}

	// ─── Casos de Uso ─────────────────────────────────────────────────────────
	createUC := usecase.NewCreateReportUseCase(reportRepo, rmq, cfg.ReportTopic)
	getUC := usecase.NewGetReportUseCase(reportRepo)

	// ─── Consumer ─────────────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go consumer.NewReportQueueConsumer(createUC, nrApp).Run(ctx, deliveries)

	// ─── HTTP (Gin) ───────────────────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(nrgin.Middleware(nrApp))

	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	internal := r.Group("/internal")
	{
		reportH := handler.NewReportHandler(getUC)
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
		log.Info().Str("port", cfg.Port).Msg("report-service started")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("shutdown error")
	}
	log.Info().Msg("report-service stopped")
}
