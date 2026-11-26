package assistant

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/logging"
)

const (
	// MaxDocumentFiles is the maximum number of documents in one upload or chat turn.
	MaxDocumentFiles = 5
	// MaxDocumentFileBytes is the maximum size of one uploaded document.
	MaxDocumentFileBytes int64 = 10 << 20
	// MaxDocumentTotalBytes is the maximum total size of one upload.
	MaxDocumentTotalBytes int64 = 50 << 20

	maxDocumentChunks       = 12
	maxDocumentContextBytes = 48 << 10
)

// AddDocuments parses and stores uploaded documents for the current user.
func (s *Service) AddDocuments(ctx context.Context, principal tool.Principal, files []*multipart.FileHeader) ([]Document, error) {
	defer logging.Trace("assistant.AddDocuments", "userId", principal.UserID, "files", len(files))()

	if s.documents == nil {
		return nil, response.ErrAIBusy()
	}
	if len(files) == 0 {
		return nil, response.ErrFileTypeInvalid()
	}
	if len(files) > MaxDocumentFiles {
		return nil, response.ErrAIProviderInvalid(fmt.Sprintf("too many documents, max %d", MaxDocumentFiles))
	}
	parsed := make([]parsedDocument, 0, len(files))
	var total int64
	for _, header := range files {
		if header == nil {
			continue
		}
		name := documentName(header.Filename)
		ext := strings.ToLower(filepath.Ext(name))
		if !supportedDocumentExt(ext) {
			return nil, response.ErrFileTypeInvalid()
		}
		file, err := header.Open()
		if err != nil {
			return nil, response.ErrFileUploadFailed(err)
		}
		data, readErr := io.ReadAll(io.LimitReader(file, MaxDocumentFileBytes+1))
		_ = file.Close()
		if readErr != nil {
			return nil, response.ErrFileUploadFailed(readErr)
		}
		if int64(len(data)) > MaxDocumentFileBytes {
			return nil, response.ErrFileTooLarge(MaxDocumentFileBytes)
		}
		total += int64(len(data))
		if total > MaxDocumentTotalBytes {
			return nil, response.ErrFileTooLarge(MaxDocumentTotalBytes)
		}
		text, err := parseDocumentText(ext, data)
		if err != nil {
			logging.ErrorS(err, "AI document parsing failed", "name", name)
			return nil, response.ErrAIDocumentInvalid(name)
		}
		text = normalizeDocumentText(text)
		if text == "" {
			return nil, response.ErrAIDocumentInvalid(name)
		}
		parsed = append(parsed, parsedDocument{
			Name:   name,
			Ext:    ext,
			Size:   int64(len(data)),
			Text:   text,
			Chunks: splitDocumentChunks(text),
		})
	}
	if len(parsed) == 0 {
		return nil, response.ErrFileTypeInvalid()
	}
	documents, err := s.documents.SaveMany(ctx, principal.UserID, parsed)
	if err != nil {
		logging.ErrorS(err, "AI document save failed")
		return nil, response.ErrFileUploadFailed(err)
	}
	return documents, nil
}

// ListDocuments returns documents owned by the current user.
func (s *Service) ListDocuments(ctx context.Context, principal tool.Principal) ([]Document, error) {
	defer logging.Trace("assistant.ListDocuments", "userId", principal.UserID)()
	if s.documents == nil {
		return nil, response.ErrAIBusy()
	}
	return s.documents.List(ctx, principal.UserID)
}

// DeleteDocument removes a document owned by the current user.
func (s *Service) DeleteDocument(ctx context.Context, principal tool.Principal, id string) error {
	defer logging.Trace("assistant.DeleteDocument", "userId", principal.UserID, "documentId", id)()
	if s.documents == nil {
		return response.ErrAIBusy()
	}
	if err := s.documents.Delete(ctx, principal.UserID, strings.TrimSpace(id)); err != nil {
		if err == sql.ErrNoRows {
			return response.ErrResourceNotFound("document")
		}
		return response.ErrFileUploadFailed(err)
	}
	return nil
}

func (s *Service) appendDocuments(ctx context.Context, principal tool.Principal, ids []string, messages []chatMessage) error {
	ids = uniqueStrings(ids)
	if len(ids) == 0 {
		return nil
	}
	if s.documents == nil {
		return response.ErrAIBusy()
	}
	if len(ids) > MaxDocumentFiles {
		return response.ErrAIProviderInvalid(fmt.Sprintf("too many documents, max %d", MaxDocumentFiles))
	}
	documents, err := s.documents.GetMany(ctx, principal.UserID, ids)
	if err != nil {
		if err == sql.ErrNoRows {
			return response.ErrResourceNotFound("document")
		}
		return response.ErrAIBusy()
	}
	last := len(messages) - 1
	query := messages[last].Content
	var chunks []documentChunk
	if !isGenericDocumentQuery(query) {
		chunks, err = s.documents.Search(ctx, documents, query, maxDocumentChunks)
		if err != nil {
			logging.ErrorS(err, "AI document search failed")
		}
	}
	if len(chunks) == 0 {
		chunks, err = s.documents.SampleChunks(ctx, documents, maxDocumentChunks)
		if err != nil {
			logging.ErrorS(err, "AI document sampling failed")
			return response.ErrAIBusy()
		}
	}
	if len(chunks) == 0 {
		return response.ErrAIDocumentInvalid("document")
	}
	var builder strings.Builder
	builder.WriteString(query)
	builder.WriteString("\n\n--- 文档上下文 ---")
	used := 0
	for _, chunk := range chunks {
		header := fmt.Sprintf("\n\n[文档: %s · 第 %d 段]\n", chunk.DocumentName, chunk.Index+1)
		if used+len(header)+len(chunk.Text) > maxDocumentContextBytes {
			break
		}
		builder.WriteString(header)
		builder.WriteString(chunk.Text)
		used += len(header) + len(chunk.Text)
	}
	if used == 0 {
		return response.ErrAIDocumentInvalid("document")
	}
	messages[last].Content = builder.String()
	return nil
}

func isGenericDocumentQuery(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.Trim(normalized, "。.!！?？")
	if normalized == "" {
		return true
	}
	for _, prefix := range []string{
		"请分析这些文件",
		"请分析这些文档",
		"分析这些文件",
		"分析这些文档",
		"总结这些文件",
		"总结这些文档",
		"please analyze these file",
		"please analyze these document",
		"analyze these file",
		"analyze these document",
		"summarize these file",
		"summarize these document",
	} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}
