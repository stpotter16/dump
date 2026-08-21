// Command backfill-embeddings re-embeds every idea that already has a
// stored embedding, using the current embedding provider. Run this once
// after switching embedding providers/models, since embeddings from
// different models are different lengths and cannot be compared.
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/stpotter16/dump/internal/store/db"
	"github.com/stpotter16/dump/internal/store/sqlite"
	"github.com/stpotter16/dump/internal/voyageai"
)

func run(ctx context.Context, getenv func(string) string) error {
	dbPath := getenv("DUMP_DB_PATH")
	if dbPath == "" {
		return errors.New("DUMP_DB_PATH environment variable not set")
	}

	voyageAPIKey := getenv("VOYAGE_API_KEY")
	if voyageAPIKey == "" {
		return errors.New("VOYAGE_API_KEY environment variable not set")
	}

	database, err := db.New(dbPath)
	if err != nil {
		return err
	}

	store, err := sqlite.New(database)
	if err != nil {
		return err
	}

	embedder := voyageai.New(voyageAPIKey)

	ideas, err := store.GetIdeas(ctx)
	if err != nil {
		return err
	}

	var reembedded, skipped int
	for _, idea := range ideas {
		if idea.Embedding == nil {
			skipped++
			continue
		}

		embedding, err := embedder.Embed(ctx, idea.Text)
		if err != nil {
			return err
		}
		if err := store.UpdateIdeaEmbedding(ctx, idea.ID, embedding); err != nil {
			return err
		}

		reembedded++
		log.Printf("Re-embedded idea %d (%d/%d)", idea.ID, reembedded, len(ideas)-skipped)

		// Stay comfortably under Voyage AI's per-minute rate limits.
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Done. Re-embedded %d ideas, skipped %d with no existing embedding.", reembedded, skipped)
	return nil
}

func main() {
	if err := run(context.Background(), os.Getenv); err != nil {
		log.Fatalf("%s", err)
	}
}
