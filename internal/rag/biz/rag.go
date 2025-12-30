// Package biz provides business logic for RAG service.
package biz

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/pkg/component/milvus"
	"github.com/kart-io/sentinel-x/pkg/component/ollama"
	"github.com/milvus-io/milvus/client/v2/entity"
)

// RAGConfig contains RAG-specific configuration.
type RAGConfig struct {
	ChunkSize    int
	ChunkOverlap int
	TopK         int
	Collection   string
	EmbeddingDim int
	DataDir      string
	SystemPrompt string
}

// RAGService provides RAG business logic.
type RAGService struct {
	milvus *milvus.Client
	ollama *ollama.Client
	cfg    *RAGConfig
}

// NewRAGService creates a new RAGService.
func NewRAGService(milvusClient *milvus.Client, ollamaClient *ollama.Client, cfg *RAGConfig) *RAGService {
	return &RAGService{
		milvus: milvusClient,
		ollama: ollamaClient,
		cfg:    cfg,
	}
}

// IndexFromURL downloads and indexes documents from a URL.
func (s *RAGService) IndexFromURL(ctx context.Context, url string) error {
	logger.Infof("Downloading documents from: %s", url)

	// Ensure data directory exists
	if err := os.MkdirAll(s.cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Download file
	zipPath := filepath.Join(s.cfg.DataDir, "docs.zip")
	if err := downloadFile(url, zipPath); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	logger.Info("Download completed")

	// Extract zip
	extractDir := filepath.Join(s.cfg.DataDir, "docs")
	if err := extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}
	logger.Info("Extraction completed")

	// Index extracted documents
	return s.IndexDirectory(ctx, extractDir)
}

// IndexDirectory indexes all documents in a directory.
func (s *RAGService) IndexDirectory(ctx context.Context, dir string) error {
	logger.Infof("Indexing documents from: %s", dir)

	// Create collection if not exists
	schema := &milvus.CollectionSchema{
		Name:        s.cfg.Collection,
		Description: "RAG knowledge base collection",
		Dimension:   s.cfg.EmbeddingDim,
		MetaFields: []milvus.MetaField{
			{Name: "document_id", DataType: entity.FieldTypeVarChar, MaxLen: 64},
			{Name: "document_name", DataType: entity.FieldTypeVarChar, MaxLen: 255},
			{Name: "section", DataType: entity.FieldTypeVarChar, MaxLen: 255},
			{Name: "content", DataType: entity.FieldTypeVarChar, MaxLen: 65535},
		},
	}
	if err := s.milvus.CreateCollection(ctx, schema); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	logger.Info("Collection ready")

	// Find all markdown files
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".mdx")) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	logger.Infof("Found %d markdown files", len(files))

	// Process files in batches
	batchSize := 10
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batch := files[i:end]

		if err := s.indexFiles(ctx, batch); err != nil {
			logger.Warnf("Failed to index batch %d-%d: %v", i, end, err)
			continue
		}
		logger.Infof("Indexed batch %d-%d", i, end)
	}

	logger.Info("Indexing completed")
	return nil
}

// indexFiles indexes a batch of files.
func (s *RAGService) indexFiles(ctx context.Context, files []string) error {
	var allChunks []chunkData

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			logger.Warnf("Failed to read file %s: %v", file, err)
			continue
		}

		// Generate document ID from file path hash
		docID := hashString(file)
		docName := filepath.Base(file)

		// Parse and chunk the content
		chunks := s.parseAndChunk(string(content), docID, docName)
		allChunks = append(allChunks, chunks...)
	}

	if len(allChunks) == 0 {
		return nil
	}

	// Generate embeddings for all chunks
	texts := make([]string, len(allChunks))
	for i, chunk := range allChunks {
		texts[i] = chunk.content
	}

	embeddings, err := s.ollama.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Prepare data for insertion
	data := &milvus.InsertData{
		Embeddings: embeddings,
		Metadata: map[string][]any{
			"document_id":   make([]any, len(allChunks)),
			"document_name": make([]any, len(allChunks)),
			"section":       make([]any, len(allChunks)),
			"content":       make([]any, len(allChunks)),
		},
	}

	for i, chunk := range allChunks {
		data.Metadata["document_id"][i] = chunk.docID
		data.Metadata["document_name"][i] = chunk.docName
		data.Metadata["section"][i] = truncateString(chunk.section, 250)
		data.Metadata["content"][i] = truncateString(chunk.content, 65000)
	}

	// Insert into Milvus
	_, err = s.milvus.Insert(ctx, s.cfg.Collection, data)
	if err != nil {
		return fmt.Errorf("failed to insert into milvus: %w", err)
	}

	return nil
}

type chunkData struct {
	docID   string
	docName string
	section string
	content string
}

// parseAndChunk parses markdown content and splits into chunks.
func (s *RAGService) parseAndChunk(content, docID, docName string) []chunkData {
	var chunks []chunkData

	// Split by headers
	headerRegex := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	sections := headerRegex.Split(content, -1)
	headers := headerRegex.FindAllStringSubmatch(content, -1)

	currentSection := "Introduction"
	for i, section := range sections {
		if i > 0 && i-1 < len(headers) {
			currentSection = headers[i-1][2]
		}

		section = strings.TrimSpace(section)
		if len(section) == 0 {
			continue
		}

		// Split section into smaller chunks
		sectionChunks := s.splitIntoChunks(section)
		for _, chunk := range sectionChunks {
			if len(strings.TrimSpace(chunk)) < 20 {
				continue // Skip very small chunks
			}
			chunks = append(chunks, chunkData{
				docID:   docID,
				docName: docName,
				section: currentSection,
				content: chunk,
			})
		}
	}

	return chunks
}

// splitIntoChunks splits text into overlapping chunks.
func (s *RAGService) splitIntoChunks(text string) []string {
	var chunks []string
	runes := []rune(text)
	chunkSize := s.cfg.ChunkSize
	overlap := s.cfg.ChunkOverlap

	if len(runes) <= chunkSize {
		return []string{text}
	}

	for i := 0; i < len(runes); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		chunks = append(chunks, chunk)
		if end == len(runes) {
			break
		}
	}

	return chunks
}

// Query performs a RAG query.
func (s *RAGService) Query(ctx context.Context, question string) (*model.QueryResult, error) {
	logger.Infof("Processing query: %s", question)

	// Generate embedding for the question
	questionEmbed, err := s.ollama.EmbedSingle(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("failed to embed question: %w", err)
	}

	// Search for similar chunks
	results, err := s.milvus.Search(ctx, s.cfg.Collection, questionEmbed, s.cfg.TopK, []string{"document_id", "document_name", "section", "content"})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(results) == 0 {
		return &model.QueryResult{
			Answer:  "I couldn't find any relevant information in the knowledge base.",
			Sources: []model.ChunkSource{},
		}, nil
	}

	// Build context from search results
	var contextBuilder strings.Builder
	sources := make([]model.ChunkSource, len(results))

	for i, result := range results {
		content := result.Metadata["content"].(string)
		docName := result.Metadata["document_name"].(string)
		section := result.Metadata["section"].(string)

		contextBuilder.WriteString(fmt.Sprintf("[%d] From %s - %s:\n%s\n\n", i+1, docName, section, content))

		sources[i] = model.ChunkSource{
			DocumentID:   result.Metadata["document_id"].(string),
			DocumentName: docName,
			Section:      section,
			Content:      content,
			Score:        result.Score,
		}
	}

	// Build prompt and generate answer
	prompt := strings.ReplaceAll(s.cfg.SystemPrompt, "{{context}}", contextBuilder.String())
	prompt = strings.ReplaceAll(prompt, "{{question}}", question)

	answer, err := s.ollama.Generate(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}

	return &model.QueryResult{
		Answer:  answer,
		Sources: sources,
	}, nil
}

// GetStats returns statistics about the knowledge base.
func (s *RAGService) GetStats(ctx context.Context) (map[string]any, error) {
	count, err := s.milvus.GetCollectionStats(ctx, s.cfg.Collection)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return map[string]any{
		"collection":    s.cfg.Collection,
		"chunk_count":   count,
		"embedding_dim": s.cfg.EmbeddingDim,
	}, nil
}

// Helper functions

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func hashString(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

func truncateString(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen])
}

// readFileContent reads file content with fallback encoding handling.
func readFileContent(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
