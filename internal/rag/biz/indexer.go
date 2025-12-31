package biz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/docutil"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/textutil"
	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// IndexerConfig 索引器配置。
type IndexerConfig struct {
	// ChunkSize 文本块大小。
	ChunkSize int
	// ChunkOverlap 块重叠大小。
	ChunkOverlap int
	// Collection 集合名称。
	Collection string
	// EmbeddingDim 嵌入向量维度。
	EmbeddingDim int
	// DataDir 数据存储目录。
	DataDir string
}

// Indexer 负责文档索引。
type Indexer struct {
	store         store.VectorStore
	embedProvider llm.EmbeddingProvider
	config        *IndexerConfig
}

// NewIndexer 创建索引器实例。
func NewIndexer(store store.VectorStore, embedProvider llm.EmbeddingProvider, config *IndexerConfig) *Indexer {
	return &Indexer{
		store:         store,
		embedProvider: embedProvider,
		config:        config,
	}
}

// IndexFromURL 从 URL 下载并索引文档。
func (i *Indexer) IndexFromURL(ctx context.Context, url string) error {
	logger.Infof("Downloading documents from: %s", url)

	if err := docutil.EnsureDir(i.config.DataDir); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	zipPath := filepath.Join(i.config.DataDir, "docs.zip")
	if err := docutil.DownloadFile(url, zipPath); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	logger.Info("Download completed")

	extractDir := filepath.Join(i.config.DataDir, "docs")
	if err := docutil.ExtractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}
	logger.Info("Extraction completed")

	return i.IndexDirectory(ctx, extractDir)
}

// IndexDirectory 索引目录中的所有文档。
func (i *Indexer) IndexDirectory(ctx context.Context, dir string) error {
	logger.Infof("Indexing documents from: %s", dir)

	collectionConfig := &store.CollectionConfig{
		Name:        i.config.Collection,
		Description: "RAG knowledge base collection",
		Dimension:   i.config.EmbeddingDim,
	}
	if err := i.store.CreateCollection(ctx, collectionConfig); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	logger.Info("Collection ready")

	files, err := docutil.FindFiles(dir, []string{".md", ".mdx"})
	if err != nil {
		return fmt.Errorf("failed to find files: %w", err)
	}

	logger.Infof("Found %d markdown files", len(files))

	batchSize := 10
	for idx := 0; idx < len(files); idx += batchSize {
		end := idx + batchSize
		if end > len(files) {
			end = len(files)
		}
		batch := files[idx:end]

		if err := i.indexFiles(ctx, batch); err != nil {
			logger.Warnf("Failed to index batch %d-%d: %v", idx, end, err)
			continue
		}
		logger.Infof("Indexed batch %d-%d", idx, end)
	}

	logger.Info("Indexing completed")
	return nil
}

// indexFiles 批量索引文件。
func (i *Indexer) indexFiles(ctx context.Context, files []string) error {
	var allChunks []*store.Chunk

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			logger.Warnf("Failed to read file %s: %v", file, err)
			continue
		}

		docID := textutil.HashString(file)
		docName := filepath.Base(file)

		chunks := i.parseAndChunk(string(content), docID, docName)
		allChunks = append(allChunks, chunks...)
	}

	if len(allChunks) == 0 {
		return nil
	}

	texts := make([]string, len(allChunks))
	for idx, chunk := range allChunks {
		texts[idx] = chunk.Content
	}

	embeddings, err := i.embedProvider.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	for idx, chunk := range allChunks {
		chunk.Embedding = embeddings[idx]
	}

	_, err = i.store.Insert(ctx, i.config.Collection, allChunks)
	return err
}

// parseAndChunk 解析并分割文档内容。
func (i *Indexer) parseAndChunk(content, docID, docName string) []*store.Chunk {
	var chunks []*store.Chunk

	headerRegex := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	sections := headerRegex.Split(content, -1)
	headers := headerRegex.FindAllStringSubmatch(content, -1)

	currentSection := "Introduction"
	for idx, section := range sections {
		if idx > 0 && idx-1 < len(headers) {
			currentSection = headers[idx-1][2]
		}

		section = strings.TrimSpace(section)
		if len(section) == 0 {
			continue
		}

		sectionChunks := textutil.SplitIntoChunks(section, i.config.ChunkSize, i.config.ChunkOverlap)
		for _, chunkContent := range sectionChunks {
			if len(strings.TrimSpace(chunkContent)) < 20 {
				continue
			}
			chunks = append(chunks, &store.Chunk{
				DocumentID:   docID,
				DocumentName: docName,
				Section:      textutil.TruncateString(currentSection, 250),
				Content:      textutil.TruncateString(chunkContent, 65000),
			})
		}
	}

	return chunks
}
