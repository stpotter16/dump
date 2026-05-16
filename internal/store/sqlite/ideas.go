package sqlite

import (
	"context"
	"encoding/binary"
	"math"
	"time"

	"github.com/stpotter16/dump/internal/types"
)

func (s Store) CreateIdea(ctx context.Context, text string, embedding []float32) (int, error) {
	now := formatTime(time.Now().UTC())
	result, err := s.db.Exec(ctx,
		`INSERT INTO idea (text, created_time, embedding) VALUES (?, ?, ?)`,
		text, now, encodeEmbedding(embedding),
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s Store) GetIdeas(ctx context.Context) ([]types.Idea, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, text, created_time, embedding FROM idea ORDER BY created_time DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ideas []types.Idea
	for rows.Next() {
		var idea types.Idea
		var createdTime string
		var embeddingBytes []byte
		if err := rows.Scan(&idea.ID, &idea.Text, &createdTime, &embeddingBytes); err != nil {
			return nil, err
		}
		idea.CreatedTime, err = parseTime(createdTime)
		if err != nil {
			return nil, err
		}
		idea.Embedding = decodeEmbedding(embeddingBytes)
		ideas = append(ideas, idea)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ideas, nil
}

func encodeEmbedding(embedding []float32) []byte {
	if embedding == nil {
		return nil
	}
	buf := make([]byte, len(embedding)*4)
	for i, f := range embedding {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func decodeEmbedding(data []byte) []float32 {
	if data == nil {
		return nil
	}
	embedding := make([]float32, len(data)/4)
	for i := range embedding {
		bits := binary.LittleEndian.Uint32(data[i*4:])
		embedding[i] = math.Float32frombits(bits)
	}
	return embedding
}
