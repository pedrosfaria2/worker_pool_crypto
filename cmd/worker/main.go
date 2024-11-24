package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pedrosfaria2/worker_pool_crypto/internal/pool"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/producer"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

const (
	numWorkers = 10
	queueSize  = 1000
)

func main() {
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	log.Printf("Conectando ao Redis em %s", redisAddr)

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal("failed to connect to redis:", err)
	}
	defer redisClient.Close()

	repo := storage.NewRedisRepository(redisClient, 24*time.Hour)
	if repo == nil {
		log.Fatal("failed to create repository")
	}

	metrics := pool.NewMetrics("worker_pool")
	workerPool := pool.NewPool(numWorkers, queueSize, metrics)

	symbols := []string{"btcusdt", "ethusdt"}
	bnc, err := producer.NewBinanceProducer(symbols, workerPool)
	if err != nil {
		log.Fatal("failed to create producer:", err)
	}

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		workerPool.Start(ctx)
	}()

	if err := bnc.Connect(ctx); err != nil {
		log.Fatal("failed to connect to binance:", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bnc.Start(ctx); err != nil {
			log.Fatal("failed to start producer:", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down gracefully...")

	cancel()

	if err := bnc.Stop(); err != nil {
		log.Println("Error stopping producer:", err)
	}

	workerPool.Stop()
	wg.Wait()

	log.Println("Shutdown complete")
}
