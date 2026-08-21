package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/stpotter16/dump/internal/handlers"
	"github.com/stpotter16/dump/internal/handlers/authentication"
	"github.com/stpotter16/dump/internal/handlers/sessions"
	"github.com/stpotter16/dump/internal/store/db"
	"github.com/stpotter16/dump/internal/store/sqlite"
	"github.com/stpotter16/dump/internal/voyageai"
)

func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	dbPath := getenv("DUMP_DB_PATH")
	if dbPath == "" {
		return errors.New("DUMP_DB_PATH environment variable not set")
	}

	voyageAPIKey := getenv("VOYAGE_API_KEY")
	if voyageAPIKey == "" {
		return errors.New("VOYAGE_API_KEY environment variable not set")
	}

	passphrase := getenv("DUMP_PASSPHRASE")
	if passphrase == "" {
		return errors.New("DUMP_PASSPHRASE environment variable not set")
	}

	log.Printf("Opening database in %v", dbPath)
	database, err := db.New(dbPath)
	if err != nil {
		return err
	}

	store, err := sqlite.New(database)
	if err != nil {
		return err
	}

	sessionManager, err := sessions.New(database, getenv)
	if err != nil {
		return err
	}

	similarityThreshold := float32(0.5)
	if raw := getenv("DUMP_SIMILARITY_THRESHOLD"); raw != "" {
		parsed, err := strconv.ParseFloat(raw, 32)
		if err != nil || parsed <= 0 || parsed > 1 {
			return fmt.Errorf("DUMP_SIMILARITY_THRESHOLD must be a number in (0, 1], got %q", raw)
		}
		similarityThreshold = float32(parsed)
	}

	authenticator := authentication.New(passphrase)
	embedderClient := voyageai.New(voyageAPIKey)

	cfg := handlers.Config{SimilarityThreshold: similarityThreshold}
	handler := handlers.NewServer(store, embedderClient, sessionManager, authenticator, cfg)
	port := getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Received termination signal. Shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(
		ctx,
		os.Args,
		os.Getenv,
		nil,
		os.Stdout,
		os.Stderr,
	); err != nil {
		log.Fatalf("%s", err)
	}
}
