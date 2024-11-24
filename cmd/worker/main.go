package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pedrosfaria2/worker_pool_crypto/internal/domain"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/pool"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/producer"
	"github.com/pedrosfaria2/worker_pool_crypto/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

const (
	numWorkers = 500
	queueSize  = 10000
)

func main() {
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	log.Printf("Connecting to Redis at %s", redisAddr)
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("failed to connect to redis:", err)
	}

	testKey := "test:connection"
	if err := redisClient.Set(ctx, testKey, "connected", time.Minute).Err(); err != nil {
		log.Fatal("failed to write to redis:", err)
	}

	val, err := redisClient.Get(ctx, testKey).Result()
	if err != nil {
		log.Fatal("failed to read from redis:", err)
	}
	log.Printf("Redis connection test successful: %s = %s", testKey, val)

	defer redisClient.Close()

	symbols, err := producer.FetchUSDTPairs(ctx)
	if err != nil {
		log.Fatal("failed to fetch trading pairs:", err)
	}
	log.Printf("Found %d USDT trading pairs", len(symbols))

	repo := storage.NewRedisRepository(redisClient, 48*time.Hour)
	if repo == nil {
		log.Fatal("failed to create repository")
	}

	metrics := pool.NewMetrics("worker_pool")
	workerPool := pool.NewPool(numWorkers, queueSize, metrics)

	bnc, err := producer.NewBinanceProducer(symbols, workerPool, repo)
	if err != nil {
		log.Fatal("failed to create producer:", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/trades", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "symbol parameter is required", http.StatusBadRequest)
			return
		}

		trades, err := repo.FindBySymbol(r.Context(), symbol, 2000)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(trades)
	})

	mux.HandleFunc("/metrics/latest", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "symbol parameter is required", http.StatusBadRequest)
			return
		}

		metrics, err := repo.GetLatestMetrics(r.Context(), symbol)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	})

	mux.HandleFunc("/metrics/all", func(w http.ResponseWriter, r *http.Request) {
		allMetrics := make(map[string]*domain.TradeMetrics)

		for _, symbol := range symbols {
			metrics, err := repo.GetLatestMetrics(r.Context(), symbol)
			if err == nil && metrics != nil {
				allMetrics[symbol] = metrics
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allMetrics)
	})

	mux.HandleFunc("/symbols", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(symbols)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Printf("HTTP server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := bnc.Stop(); err != nil {
		log.Printf("Error stopping producer: %v", err)
	}

	workerPool.Stop()
	wg.Wait()

	log.Println("Shutdown complete")
}
