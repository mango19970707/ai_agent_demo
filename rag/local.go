package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/embedding"
)

// LocalRAGConfig 配置本地存储
type LocalRAGConfig struct {
	// FilePath 为本地向量数据保存路径（JSON 格式）
	FilePath string
	// Dimensions 为向量维度（用于校验）
	Dimensions int
}

// localDoc 表示本地存储的向量条目
type localDoc struct {
	ID       string            `json:"id"`
	Content  string            `json:"content"`
	Vector   []float32         `json:"vector"`
	Metadata map[string]string `json:"metadata"`
}

// LocalRAG 提供在本地文件上的导入与检索能力
type LocalRAG struct {
	cfg      *LocalRAGConfig
	mu       sync.RWMutex
	loaded   bool
	inMemory map[string]*localDoc // 以 ID 为键
}

// NewLocalRAG 初始化本地 RAG 存储，若文件存在则加载
func NewLocalRAG(ctx context.Context, cfg *LocalRAGConfig) (*LocalRAG, error) {
	if cfg == nil || cfg.FilePath == "" {
		return nil, errors.New("local rag config invalid: missing FilePath")
	}
	lr := &LocalRAG{
		cfg:      cfg,
		inMemory: make(map[string]*localDoc),
	}
	if err := lr.loadFromDisk(); err != nil {
		return nil, err
	}
	return lr, nil
}

// ImportDocument 将文本嵌入并写入本地文件（追加/更新）
func (l *LocalRAG) ImportDocument(ctx context.Context, content, docType, source string, metadata map[string]string, emb embedding.Embedder) error {
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
	if l.cfg.Dimensions > 0 && len(vectors[0]) != l.cfg.Dimensions {
		log.Printf("warning: vector dims %d != config %d", len(vectors[0]), l.cfg.Dimensions)
	}

	// 转换向量类型到 float32
	vec32 := make([]float32, len(vectors[0]))
	for i, v := range vectors[0] {
		vec32[i] = float32(v)
	}

	// 准备元数据
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["doc_type"] = docType
	metadata["source"] = source
	metadata["import_time"] = time.Now().Format("2006-01-02 15:04:05")

	// 写入内存并落盘
	l.mu.Lock()
	l.inMemory[docID] = &localDoc{ID: docID, Content: content, Vector: vec32, Metadata: metadata}
	l.mu.Unlock()

	return l.flushToDisk()
}

// SearchTopK 对查询进行检索（采用 Cosine 相似度，内存扫描）
func (l *LocalRAG) SearchTopK(ctx context.Context, queries []string, topK int, emb embedding.Embedder) ([]DocChunk, error) {
	if emb == nil {
		return nil, errors.New("embedder is nil")
	}
	if topK <= 0 {
		topK = 5
	}

	l.mu.RLock()
	docs := make([]*localDoc, 0, len(l.inMemory))
	for _, d := range l.inMemory {
		docs = append(docs, d)
	}
	l.mu.RUnlock()

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

		// 按相似度打分
		scored := make([]struct {
			chunk DocChunk
			score float64
		}, 0, len(docs))
		for _, d := range docs {
			if len(d.Vector) == 0 {
				continue
			}
			if len(d.Vector) != len(qv) {
				// 尺寸不一致时跳过
				continue
			}
			s := CosineSimilarity(d.Vector, qv)
			scored = append(scored, struct {
				chunk DocChunk
				score float64
			}{
				chunk: DocChunk{ID: d.ID, Content: d.Content, Score: s, Metadata: d.Metadata},
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

// loadFromDisk 加载本地 JSON 文件
func (l *LocalRAG) loadFromDisk() error {
	f, err := os.Open(l.cfg.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 不存在则创建一个空文件
			if err := os.WriteFile(l.cfg.FilePath, []byte("[]"), 0644); err != nil {
				return fmt.Errorf("create empty local store: %w", err)
			}
			l.loaded = true
			return nil
		}
		return fmt.Errorf("open local store: %w", err)
	}
	defer f.Close()
	var arr []localDoc
	dec := json.NewDecoder(f)
	if err := dec.Decode(&arr); err != nil {
		return fmt.Errorf("decode local store: %w", err)
	}
	l.mu.Lock()
	for i := range arr {
		d := arr[i]
		// 防御：nil metadata
		if d.Metadata == nil {
			d.Metadata = map[string]string{}
		}
		l.inMemory[d.ID] = &d
	}
	l.mu.Unlock()
	l.loaded = true
	return nil
}

// flushToDisk 将内存数据写回文件
func (l *LocalRAG) flushToDisk() error {
	l.mu.RLock()
	arr := make([]localDoc, 0, len(l.inMemory))
	for _, d := range l.inMemory {
		arr = append(arr, *d)
	}
	l.mu.RUnlock()
	bs, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal local store: %w", err)
	}
	if err := os.WriteFile(l.cfg.FilePath, bs, 0644); err != nil {
		return fmt.Errorf("write local store: %w", err)
	}
	return nil
}

// cosine 计算余弦相似度
func (l *LocalRAG) cosine(a, b []float32) float64 {
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
