// Package model provides data models for the Sentinel-X platform.
package model

import (
	"time"
)

// Document represents a document in the knowledge base.
type Document struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	Title     string    `json:"title" gorm:"type:varchar(255);not null"`
	Source    string    `json:"source" gorm:"type:varchar(512);not null"` // File path or URL
	Content   string    `json:"content,omitempty" gorm:"type:longtext"`
	Hash      string    `json:"hash" gorm:"type:varchar(64);index"` // Content hash for deduplication
	ChunkNum  int       `json:"chunk_num" gorm:"default:0"`
	Status    string    `json:"status" gorm:"type:varchar(32);default:'pending'"` // pending, indexed, failed
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for Document.
func (Document) TableName() string {
	return "rag_documents"
}

// Chunk represents a text chunk in the knowledge base.
type Chunk struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	DocumentID string    `json:"document_id" gorm:"type:varchar(64);index;not null"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	Section    string    `json:"section" gorm:"type:varchar(255)"` // Section/heading
	StartPos   int       `json:"start_pos" gorm:"default:0"`
	EndPos     int       `json:"end_pos" gorm:"default:0"`
	VectorID   int64     `json:"vector_id" gorm:"index"` // ID in Milvus
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for Chunk.
func (Chunk) TableName() string {
	return "rag_chunks"
}

// QueryResult represents a RAG query result.
type QueryResult struct {
	Answer  string        `json:"answer"`
	Sources []ChunkSource `json:"sources"`
}

// ChunkSource represents source information for a retrieved chunk.
type ChunkSource struct {
	DocumentID   string  `json:"document_id"`
	DocumentName string  `json:"document_name"`
	Section      string  `json:"section"`
	Content      string  `json:"content"`
	Score        float32 `json:"score"`
}
