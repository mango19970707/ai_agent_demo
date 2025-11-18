package rag

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"math"
	"time"

	"github.com/cloudwego/eino/components/embedding"
)

// VectorStore 抽象统一的向量存储接口，便于依赖注入与实现切换。
// 所有存储实现（MilvusRAG/LocalRAG/RedisRAG）均应满足该接口。
type VectorStore interface {
	// ImportDocument 将文本嵌入后写入存储。
	// - content: 文本内容
	// - docType: 文档类型（如“novel”/“faq”）
	// - source: 文档来源（如文件名/URL）
	// - metadata: 额外元数据（可为 nil）
	// - emb: 向量化组件
	ImportDocument(ctx context.Context, content, docType, source string, metadata map[string]string, emb embedding.Embedder) error

	// SearchTopK 基于查询检索相似内容。
	// - queries: 多条查询语句
	// - topK: 每条查询返回的上限
	// - emb: 向量化组件
	SearchTopK(ctx context.Context, queries []string, topK int, emb embedding.Embedder) ([]DocChunk, error)
}

// GenerateDocID 基于内容/类型/来源生成稳定的文档 ID（带时间戳避免碰撞）。
func GenerateDocID(content, docType, source string) string {
	h := md5.Sum([]byte(content + docType + source + time.Now().Format("20060102150405")))
	return hex.EncodeToString(h[:])
}

// BuildMetadata 统一构造元数据，保留用户传入并补充标准字段。
func BuildMetadata(docType, source string, metadata map[string]string) map[string]string {
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["doc_type"] = docType
	metadata["source"] = source
	metadata["import_time"] = time.Now().Format("2006-01-02 15:04:05")
	return metadata
}

// CosineSimilarity 计算余弦相似度（越大越相似）。
func CosineSimilarity(a, b []float32) float64 {
	var dot float64
	var na, nb float64
	for i := 0; i < len(a) && i < len(b); i++ {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		na += af * af
		nb += bf * bf
	}
	den := math.Sqrt(na) * math.Sqrt(nb)
	if den == 0 {
		return 0
	}
	return dot / den
}