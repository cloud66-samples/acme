package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-redis/redis"
)

type key int

const (
	interval         = 1 * time.Second
	keyName          = "acme:queue"
	histogram        = "acme:histogram"
	requestIDKey key = 0
	MaxDiv           = 5
	SeedBase         = 200
)

var (
	redisAddress string
	client       *redis.Client
	listenAddr   string
	healthy      int32
)

// this pushes new items onto a stack on a random cycle
func main() {
	flag.StringVar(&redisAddress, "redis", "localhost:6379", "Connection to redis")
	flag.StringVar(&listenAddr, "binding", "0.0.0.0:5000", "Server listen address")

	flag.Parse()

	cancel := make(chan os.Signal)
	signal.Notify(cancel, os.Interrupt, syscall.SIGTERM)

	rand.Seed(time.Now().Unix())

	err := setupClient()
	if err != nil {
		fmt.Printf("Failed to connect to Redis on %s due to %s\n", redisAddress, err.Error())
		os.Exit(1)
	}

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Printf("Server is starting on %s...\n", listenAddr)

	router := http.NewServeMux()
	router.Handle("/", http.FileServer(http.Dir("./static")))
	router.Handle("/size", sizeRoute())
	router.Handle("/histogram", histogramRoute())
	router.Handle("/healthz", healthz())

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      tracing(nextRequestID)(logging(logger)(router)),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	seed := rand.Float64() + SeedBase

	go func() {
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-quit:
				fmt.Println("Leaving...")
				ticker.Stop()
				os.Exit(1)
			case <-ticker.C:
				fmt.Println("Getting tickers...")
				dp, close := createDatapoint(seed)
				seed = close
				dpJSON, err := json.Marshal(dp)
				if err != nil {
					fmt.Printf("Failed to serialise tickers due to %s\n", err.Error())
				}
				_, err = client.LPush(histogram, dpJSON).Result()
				if err != nil {
					fmt.Printf("Failed to write tickers due to %s\n", err.Error())
				}
			}
		}
	}()

	go func() {
		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	logger.Println("Server stopped")
}

// seed is last tick's close
func createDatapoint(seed float64) ([]float64, float64) {
	t := time.Now().Unix()
	open := getValue(seed)
	close := getValue(seed)
	high := getHigh(open, close)
	low := getLow(open, close)

	return []float64{float64(t), open, close, high, low}, close
}

func getDir() float64 {
	dir := rand.Float64()
	if dir > 0.5 {
		return 1
	} else {
		return -1
	}
}

func getHigh(open, close float64) float64 {
	day := []float64{open, close, getValue(open)}
	max := 0.0
	for _, n := range day {
		max = math.Max(max, n)
	}

	return max
}

func getLow(open, close float64) float64 {
	day := []float64{open, close, getValue(open)}
	min := 99999999.00
	for _, n := range day {
		min = math.Min(min, n)
	}

	return min
}

func getValue(seed float64) float64 {
	return seed + (getDir() * getDiv())
}

func getDiv() float64 {
	return float64(rand.Intn(MaxDiv))
}

func setupClient() error {
	client = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "",
		DB:       0,
	})

	fmt.Printf("Connecting to Redis on %s\n", redisAddress)
	_, err := client.Ping().Result()
	return err
}

func fetch() (int64, error) {
	size, err := client.LLen(keyName).Result()
	if err != nil {
		return 0, err
	}

	return size, nil
}

func sizeRoute() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		size, _ := fetch()
		fmt.Fprintf(w, "{ \"size\": %d }", size)
	})
}

func histogramRoute() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		ticker, _ := client.RPop(histogram).Result()
		var day []float64
		fmt.Printf("Ticker %s\n", ticker)
		err := json.Unmarshal([]byte(ticker), &day)
		if err != nil {
			fmt.Printf("Failed to unmarshal ticker due to %s\n", err.Error())
		}
		json.NewEncoder(w).Encode(day)
	})
}

func healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
