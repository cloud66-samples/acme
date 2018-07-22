package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis"
)

const (
	interval = 1 * time.Second
	keyName  = "acme:queue"
)

var (
	redisAddress string
	client       *redis.Client
)

// this pushes new items onto a stack on a random cycle
func main() {
	flag.StringVar(&redisAddress, "redis", "localhost:6379", "Connection to redis")
	flag.Parse()

	cancel := make(chan os.Signal)
	signal.Notify(cancel, os.Interrupt, syscall.SIGTERM)

	err := setupClient()
	if err != nil {
		fmt.Printf("Failed to connect to Redis on %s due to %s\n", redisAddress, err.Error())
		os.Exit(1)
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-cancel:
			fmt.Println("Leaving...")
			ticker.Stop()
			os.Exit(1)
		case <-ticker.C:
			fmt.Println("Buying...")
			err := buy()
			if err != nil {
				fmt.Println("Nothing to buy")
			}
		}
	}
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

func buy() error {
	result, err := client.LPop(keyName).Result()
	if err != nil {
		return err
	}

	fmt.Println(result)

	return nil
}
