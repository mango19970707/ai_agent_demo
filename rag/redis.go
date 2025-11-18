package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	redis "github.com/redis/go-redis/v9"
)

// RedisRAGConfig 配置 Redis 存储
type RedisRAGConfig struct {
	Address    string // 例如: 127.0.0.1:6379
	Password   string
	DB         int
	Prefix     string // key 前缀，默认 "rag"
	Dimensions int
}

// RedisRAG 通过 Redis 存储向量与元数据
// 说明：本实现使用应用侧全量扫描计算相似度，适合数据量较小的演示/开发环境。
type RedisRAG struct {
	cli  *redis.Client
	cfg  *RedisRAGConfig
	pref string
}

// NewRedisRAG 初始化 Redis 客户端
func NewRedisRAG(ctx context.Context, cfg *RedisRAGConfig) (*RedisRAG, error) {
	if cfg == nil || cfg.Address == "" {
		return nil, errors.New("redis rag config invalid: missing Address")
	}
	pref := cfg.Prefix
	if pref == "" {
		pref = "rag"
	}
	cli := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := cli.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisRAG{cli: cli, cfg: cfg, pref: pref}, nil
}

// ImportDocument 将文本嵌入并写入 Redis
func (r *RedisRAG) ImportDocument(ctx context.Context, content, docType, source string, metadata map[string]string, emb embedding.Embedder) error {
	if emb == nil {
		return errors.New("embedder is nil")
	}
	// 生成文档ID（使用统一助手）
	docID := GenerateDocID(content, docType, source)

	// 生成向量
	vectors, err := emb.EmbedStrings(ctx, []string{content})
	if err != nil {
		return fmt.Errorf("generate embedding: %w", err)
	}
	if len(vectors) == 0 {
		return errors.New("no vectors generated")
	}
	if r.cfg.Dimensions > 0 && len(vectors[0]) != r.cfg.Dimensions {
		log.Printf("warning: vector dims %d != config %d", len(vectors[0]), r.cfg.Dimensions)
	}

	// 转 float32 并序列化为 JSON
	vec32 := make([]float32, len(vectors[0]))
	for i, v := range vectors[0] {
		vec32[i] = float32(v)
	}
	vecJSON, err := json.Marshal(vec32)
	if err != nil {
		return fmt.Errorf("marshal vector: %w", err)
	}

	// 元数据
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["doc_type"] = docType
	metadata["source"] = source
	metadata["import_time"] = time.Now().Format("2006-01-02 15:04:05")
	metadata = BuildMetadata(docType, source, metadata)
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	key := fmt.Sprintf("%s:doc:%s", r.pref, docID)

	// 使用哈希存储文档
	if err := r.cli.HSet(ctx, key, map[string]any{
		"id":       docID,
		"content":  content,
		"vector":   string(vecJSON),
		"metadata": string(metaJSON),
	}).Err(); err != nil {
		return fmt.Errorf("redis hset: %w", err)
	}

	// 把文档 ID 纳入集合，便于枚举
	if err := r.cli.SAdd(ctx, fmt.Sprintf("%s:docs", r.pref), docID).Err(); err != nil {
		return fmt.Errorf("redis sadd: %w", err)
	}
	return nil
}

// SearchTopK 在应用侧枚举所有文档并计算相似度（Cosine）
func (r *RedisRAG) SearchTopK(ctx context.Context, queries []string, topK int, emb embedding.Embedder) ([]DocChunk, error) {
	if emb == nil {
		return nil, errors.New("embedder is nil")
	}
	if topK <= 0 {
		topK = 5
	}
	ids, err := r.cli.SMembers(ctx, fmt.Sprintf("%s:docs", r.pref)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis smembers: %w", err)
	}

	var out []DocChunk
	for _, q := range queries {
		vecs, err := emb.EmbedStrings(ctx, []string{q})
		if err != nil {
			return nil, fmt.Errorf("generate query embedding: %w", err)
		}
		if len(vecs) == 0 {
			continue
		}
		qv := make([]float32, len(vecs[0]))
		for i, v := range vecs[0] {
			qv[i] = float32(v)
		}

		scored := make([]struct {
			chunk DocChunk
			score float64
		}, 0, len(ids))

		for _, id := range ids {
			key := fmt.Sprintf("%s:doc:%s", r.pref, id)
			vals, err := r.cli.HGetAll(ctx, key).Result()
			if err != nil {
				log.Printf("hgetall %s error: %v", key, err)
				continue
			}
			if len(vals) == 0 {
				continue
			}
			var vec32 []float32
			if err := json.Unmarshal([]byte(vals["vector"]), &vec32); err != nil {
				log.Printf("vector unmarshal %s error: %v", key, err)
				continue
			}
			if len(vec32) != len(qv) {
				continue
			}
			var meta map[string]string
			_ = json.Unmarshal([]byte(vals["metadata"]), &meta)
			s := CosineSimilarity(vec32, qv)
			scored = append(scored, struct {
				chunk DocChunk
				score float64
			}{
				chunk: DocChunk{ID: id, Content: vals["content"], Score: s, Metadata: meta},
				score: s,
			})
		}

		sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
		limit := topK
		if limit > len(scored) {
			limit = len(scored)
		}
		for i := 0; i < limit; i++ {
			out = append(out, scored[i].chunk)
		}
	}
	return out, nil
}

// 已迁移至通用助手 CosineSimilarity(a, b []float32)
