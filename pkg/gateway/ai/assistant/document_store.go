package assistant

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/oklog/ulid/v2"
)

const (
	documentsTableSQL = `
CREATE TABLE IF NOT EXISTS ai_documents (
    id             TEXT PRIMARY KEY,
    owner_user_id  TEXT NOT NULL,
    name           TEXT NOT NULL,
    ext            TEXT NOT NULL,
    size           INTEGER NOT NULL,
    text_size      INTEGER NOT NULL,
    chunk_count    INTEGER NOT NULL,
    created_time   TEXT NOT NULL
);`
	documentsIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_ai_documents_owner_created
ON ai_documents (owner_user_id, created_time DESC);`
	documentChunksTableSQL = `
CREATE TABLE IF NOT EXISTS ai_document_chunks (
    document_id   TEXT NOT NULL,
    chunk_index   INTEGER NOT NULL,
    text          TEXT NOT NULL,
    PRIMARY KEY (document_id, chunk_index)
);`
	documentChunksFTSSQL = `
CREATE VIRTUAL TABLE IF NOT EXISTS ai_document_chunks_fts USING fts5(
    document_id UNINDEXED,
    chunk_index UNINDEXED,
    text,
    tokenize='trigram'
);`
)

// Document is an uploaded long document that can be referenced by the assistant.
type Document struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Ext         string    `json:"ext"`
	Size        int64     `json:"size"`
	TextSize    int       `json:"textSize"`
	ChunkCount  int       `json:"chunkCount"`
	CreatedTime time.Time `json:"createdTime"`
}

type parsedDocument struct {
	Name   string
	Ext    string
	Size   int64
	Text   string
	Chunks []string
}

type documentChunk struct {
	DocumentID   string
	DocumentName string
	Index        int
	Text         string
}

// DocumentStore persists parsed documents and their retrieval chunks in SQLite.
type DocumentStore struct {
	db *sql.DB
}

// NewDocumentStore creates a document store backed by the gateway SQLite database.
func NewDocumentStore(db *sql.DB) *DocumentStore {
	return &DocumentStore{db: db}
}

// Initialize creates the document tables and full-text index.
func (s *DocumentStore) Initialize(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("document store database is nil")
	}
	for _, statement := range []string{
		documentsTableSQL,
		documentsIndexSQL,
		documentChunksTableSQL,
		documentChunksFTSSQL,
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

// SaveMany stores parsed documents and chunks atomically.
func (s *DocumentStore) SaveMany(ctx context.Context, ownerID string, items []parsedDocument) ([]Document, error) {
	if len(items) == 0 {
		return []Document{}, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()
	result := make([]Document, 0, len(items))
	for _, item := range items {
		id := ulid.Make().String()
		document := Document{
			ID:          id,
			Name:        item.Name,
			Ext:         item.Ext,
			Size:        item.Size,
			TextSize:    len(item.Text),
			ChunkCount:  len(item.Chunks),
			CreatedTime: now,
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO ai_documents
			 (id, owner_user_id, name, ext, size, text_size, chunk_count, created_time)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			document.ID, ownerID, document.Name, document.Ext, document.Size,
			document.TextSize, document.ChunkCount, document.CreatedTime.Format(time.RFC3339Nano),
		); err != nil {
			return nil, err
		}
		for index, chunk := range item.Chunks {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO ai_document_chunks (document_id, chunk_index, text) VALUES (?, ?, ?)`,
				document.ID, index, chunk,
			); err != nil {
				return nil, err
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO ai_document_chunks_fts (document_id, chunk_index, text) VALUES (?, ?, ?)`,
				document.ID, index, chunk,
			); err != nil {
				return nil, err
			}
		}
		result = append(result, document)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// List returns documents owned by the user.
func (s *DocumentStore) List(ctx context.Context, ownerID string) ([]Document, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, ext, size, text_size, chunk_count, created_time
		 FROM ai_documents WHERE owner_user_id = ? ORDER BY created_time DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Document, 0)
	for rows.Next() {
		item, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

// GetMany returns documents owned by the user in the requested order.
func (s *DocumentStore) GetMany(ctx context.Context, ownerID string, ids []string) ([]Document, error) {
	ids = uniqueStrings(ids)
	if len(ids) == 0 {
		return []Document{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+1)
	args = append(args, ownerID)
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, ext, size, text_size, chunk_count, created_time
		 FROM ai_documents WHERE owner_user_id = ? AND id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := make(map[string]Document, len(ids))
	for rows.Next() {
		item, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		found[item.ID] = *item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(found) != len(ids) {
		return nil, sql.ErrNoRows
	}
	result := make([]Document, 0, len(ids))
	for _, id := range ids {
		result = append(result, found[id])
	}
	return result, nil
}

// Delete removes a document and all of its chunks.
func (s *DocumentStore) Delete(ctx context.Context, ownerID string, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	if err := tx.QueryRowContext(ctx,
		`SELECT 1 FROM ai_documents WHERE id = ? AND owner_user_id = ?`, id, ownerID,
	).Scan(&exists); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ai_document_chunks_fts WHERE document_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ai_document_chunks WHERE document_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ai_documents WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// Search finds the chunks most relevant to the query across the given documents.
func (s *DocumentStore) Search(ctx context.Context, documents []Document, query string, limit int) ([]documentChunk, error) {
	if len(documents) == 0 || limit <= 0 {
		return nil, nil
	}
	ftsTerms, likeTerms := retrievalTerms(query)
	if len(ftsTerms) > 0 {
		items, err := s.searchFTS(ctx, documents, ftsTerms, limit)
		if err == nil && len(items) > 0 {
			return items, nil
		}
	}
	if len(likeTerms) == 0 {
		return nil, nil
	}
	return s.searchLike(ctx, documents, likeTerms, limit)
}

// SampleChunks returns chunks spread across the documents for broad questions.
func (s *DocumentStore) SampleChunks(ctx context.Context, documents []Document, limit int) ([]documentChunk, error) {
	if len(documents) == 0 || limit <= 0 {
		return nil, nil
	}
	perDocument := limit / len(documents)
	if perDocument < 1 {
		perDocument = 1
	}
	result := make([]documentChunk, 0, limit)
	for _, document := range documents {
		indexes := sampleChunkIndexes(document.ChunkCount, perDocument)
		if len(indexes) == 0 {
			continue
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(indexes)), ",")
		args := make([]any, 0, len(indexes)+1)
		args = append(args, document.ID)
		for _, index := range indexes {
			args = append(args, index)
		}
		rows, err := s.db.QueryContext(ctx,
			`SELECT c.document_id, c.chunk_index, c.text, d.name
			 FROM ai_document_chunks c
			 JOIN ai_documents d ON d.id = c.document_id
			 WHERE c.document_id = ? AND c.chunk_index IN (`+placeholders+`)
			 ORDER BY c.chunk_index`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var item documentChunk
			if err := rows.Scan(&item.DocumentID, &item.Index, &item.Text, &item.DocumentName); err != nil {
				_ = rows.Close()
				return nil, err
			}
			result = append(result, item)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		if len(result) >= limit {
			break
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *DocumentStore) searchFTS(ctx context.Context, documents []Document, terms []string, limit int) ([]documentChunk, error) {
	ids := make([]string, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document.ID)
	}
	query := make([]string, 0, len(terms))
	for _, term := range terms {
		query = append(query, `"`+strings.ReplaceAll(term, `"`, `""`)+`"`)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+2)
	args = append(args, strings.Join(query, " OR "))
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx,
		`SELECT f.document_id, f.chunk_index, f.text, d.name
		 FROM ai_document_chunks_fts f
		 JOIN ai_documents d ON d.id = f.document_id
		 WHERE f.text MATCH ? AND f.document_id IN (`+placeholders+`)
		 ORDER BY bm25(f)
		 LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]documentChunk, 0, limit)
	for rows.Next() {
		var item documentChunk
		if err := rows.Scan(&item.DocumentID, &item.Index, &item.Text, &item.DocumentName); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *DocumentStore) searchLike(ctx context.Context, documents []Document, terms []string, limit int) ([]documentChunk, error) {
	ids := make([]string, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document.ID)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	items := make([]documentChunk, 0, limit)
	seen := make(map[string]bool, limit)
	for _, term := range terms {
		args := make([]any, 0, len(ids)+2)
		args = append(args, "%"+escapeLike(term)+"%")
		for _, id := range ids {
			args = append(args, id)
		}
		args = append(args, limit)
		rows, err := s.db.QueryContext(ctx,
			`SELECT f.document_id, f.chunk_index, f.text, d.name
			 FROM ai_document_chunks f
			 JOIN ai_documents d ON d.id = f.document_id
			 WHERE f.text LIKE ? ESCAPE '\' AND f.document_id IN (`+placeholders+`)
			 LIMIT ?`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var item documentChunk
			if err := rows.Scan(&item.DocumentID, &item.Index, &item.Text, &item.DocumentName); err != nil {
				_ = rows.Close()
				return nil, err
			}
			key := item.DocumentID + ":" + fmt.Sprint(item.Index)
			if seen[key] {
				continue
			}
			seen[key] = true
			items = append(items, item)
			if len(items) >= limit {
				_ = rows.Close()
				return items, nil
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	return items, nil
}

func scanDocument(scanner interface{ Scan(...any) error }) (*Document, error) {
	var item Document
	var created string
	if err := scanner.Scan(&item.ID, &item.Name, &item.Ext, &item.Size,
		&item.TextSize, &item.ChunkCount, &created); err != nil {
		return nil, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	item.CreatedTime = parsed
	return &item, nil
}

func retrievalTerms(value string) ([]string, []string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	ftsTerms := make([]string, 0)
	likeTerms := make([]string, 0)
	seenFTS := make(map[string]bool)
	seenLike := make(map[string]bool)
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		run := strings.TrimSpace(strings.ToLower(string(current)))
		current = nil
		runes := []rune(run)
		if len(runes) >= 3 {
			for index := 0; index+3 <= len(runes); index++ {
				term := string(runes[index : index+3])
				if !seenFTS[term] && len(ftsTerms) < 32 {
					seenFTS[term] = true
					ftsTerms = append(ftsTerms, term)
				}
			}
		}
		if len(runes) >= 2 {
			if !seenLike[run] && len(likeTerms) < 16 {
				seenLike[run] = true
				likeTerms = append(likeTerms, run)
			}
			if unicode.Is(unicode.Han, runes[0]) {
				for index := 0; index+2 <= len(runes); index++ {
					term := string(runes[index : index+2])
					if !seenLike[term] && len(likeTerms) < 16 {
						seenLike[term] = true
						likeTerms = append(likeTerms, term)
					}
				}
			}
		}
	}
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if len(current) > 0 && unicode.Is(unicode.Han, char) != unicode.Is(unicode.Han, current[len(current)-1]) {
				flush()
			}
			current = append(current, char)
			continue
		}
		flush()
	}
	flush()
	return ftsTerms, likeTerms
}

func sampleChunkIndexes(count int, limit int) []int {
	if count <= 0 || limit <= 0 {
		return nil
	}
	if count <= limit {
		result := make([]int, count)
		for index := range result {
			result[index] = index
		}
		return result
	}
	if limit == 1 {
		return []int{0}
	}
	result := make([]int, 0, limit)
	for index := 0; index < limit; index++ {
		position := index * (count - 1) / (limit - 1)
		if len(result) == 0 || result[len(result)-1] != position {
			result = append(result, position)
		}
	}
	return result
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}
