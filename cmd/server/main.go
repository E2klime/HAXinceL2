package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/E2klime/HAXinceL2/internal"
	"github.com/E2klime/HAXinceL2/telegram"
)

func main() {
	addr := flag.String("addr", ":8080", "Server address")
	botToken := flag.String("bot-token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token")
	adminIDs := flag.String("admin-ids", os.Getenv("TELEGRAM_ADMIN_IDS"), "Comma-separated list of admin Telegram IDs")
	flag.Parse()

	if *botToken == "" {
		log.Fatal("Telegram bot token is required (use -bot-token or TELEGRAM_BOT_TOKEN env)")
	}

	if *adminIDs == "" {
		log.Fatal("Admin IDs are required (use -admin-ids or TELEGRAM_ADMIN_IDS env)")
	}

	parsedAdminIDs, err := telegram.ParseAdminIDs(*adminIDs)
	if err != nil {
		log.Fatalf("Failed to parse admin IDs: %v", err)
	}

	srv := internal.NewServer()
	go srv.Run()

	bot, err := telegram.NewBot(*botToken, srv, parsedAdminIDs)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	go bot.Start()

	http.HandleFunc("/ws", srv.HandleWebSocket)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Server started on %s", *addr)
	log.Printf("Telegram bot started with %d admin(s)", len(parsedAdminIDs))

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
