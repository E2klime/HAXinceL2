package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/E2klime/HAXinceL2/internal"
)

func main() {
	serverURL := flag.String("server", os.Getenv("SERVER_URL"), "Server WebSocket URL (e.g., ws://server.com:8080/ws)")
	flag.Parse()

	if *serverURL == "" {
		log.Fatal("Server URL is required (use -server or SERVER_URL env)")
	}

	c, err := internal.NewClient(*serverURL)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		err := c.Connect()
		if err != nil {
			log.Printf("Failed to connect: %v. Retrying in 10 seconds...", err)
			time.Sleep(10 * time.Second)
			continue
		}

		go func() {
			if err := c.Run(); err != nil {
				log.Printf("Client error: %v. Reconnecting...", err)
				time.Sleep(5 * time.Second)
			}
		}()

		break
	}

	<-sigChan
	log.Println("Shutting down...")
}
