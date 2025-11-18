package rag

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino-ext/components/retriever/milvus"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

type MilvusRAGConfig struct {
	Address    string
	Username   string
	Password   string
	Dimensions int
	Collection string
}

// MilvusRAG 封装 Milvus 检索逻辑，负责将查询语句转为向量并从 Milvus 拿回相似片段。
type MilvusRAG struct {
	ret *milvus.Retriever
	cli client.Client
	cfg *MilvusRAGConfig
}

// DocChunk 表示从向量数据库检索得到的片段。
type DocChunk struct {
	ID       string            // 唯一 ID
	Score    float64           // 相似度分数
	Content  string            // 片段正文
	Metadata map[string]string // 元数据，如来源、角色、章节等
}

// NewMilvusRAG 创建检索器，需要提供连接配置和 Embedder。
// 如果集合不存在，会尝试创建一个简单的集合用于测试。
func NewMilvusRAG(ctx context.Context, cfg *MilvusRAGConfig, emb embedding.Embedder) (*MilvusRAG, error) {
	cli, err := client.NewClient(ctx, client.Config{
		Address:  cfg.Address,
		Username: cfg.Username,
		Password: cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("milvus client: %w", err)
	}

	// 获取嵌入维度
	dim := cfg.Dimensions
	if dim <= 0 {
		dim = 1536 // 默认维度
	}

	// 检查集合是否存在
	has, err := cli.HasCollection(ctx, cfg.Collection)
	if err != nil {
		return nil, fmt.Errorf("check collection: %w", err)
	}

	// 如果集合存在，检查维度是否匹配
	if has {
		// 获取集合信息
		desc, err := cli.DescribeCollection(ctx, cfg.Collection)
		if err != nil {
			return nil, fmt.Errorf("describe collection: %w", err)
		}

		// 检查向量字段的维度
		var vectorDim int
		for _, field := range desc.Schema.Fields {
			if field.Name == "vector" && field.DataType == entity.FieldTypeFloatVector {
				if dimStr, ok := field.TypeParams["dim"]; ok {
					fmt.Sscanf(dimStr, "%d", &vectorDim)
				}
				break
			}
		}

		// 如果维度不匹配，删除并重新创建集合
		if vectorDim != dim {
			log.Printf("集合 %s 的向量维度 %d 与配置维度 %d 不匹配，删除并重新创建...", cfg.Collection, vectorDim, dim)

			// 释放集合
			err = cli.ReleaseCollection(ctx, cfg.Collection)
			if err != nil {
				log.Printf("释放集合失败（可能已释放）: %v", err)
			}

			// 删除集合
			err = cli.DropCollection(ctx, cfg.Collection)
			if err != nil {
				return nil, fmt.Errorf("drop collection: %w", err)
			}
			has = false // 标记为不存在，需要重新创建
		} else {
			log.Printf("集合 %s 已存在且维度匹配 (%d)", cfg.Collection, dim)
		}
	}

	// 如果集合不存在，创建一个新集合
	if !has {
		log.Printf("集合 %s 不存在，创建新集合（维度: %d）...", cfg.Collection, dim)

		// 创建集合
		schema := &entity.Schema{
			CollectionName: cfg.Collection,
			Description:    "Novel knowledge base for RAG",
			Fields: []*entity.Field{
				{
					Name:       "id",
					DataType:   entity.FieldTypeVarChar,
					PrimaryKey: true,
					AutoID:     false,
					TypeParams: map[string]string{
						"max_length": "100",
					},
				},
				{
					Name:     "content",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						"max_length": "65535",
					},
				},
				{
					Name:     "vector",
					DataType: entity.FieldTypeFloatVector,
					TypeParams: map[string]string{
						"dim": fmt.Sprintf("%d", dim),
					},
				},
				{
					Name:     "metadata",
					DataType: entity.FieldTypeJSON,
				},
			},
		}

		err = cli.CreateCollection(ctx, schema, 1)
		if err != nil {
			return nil, fmt.Errorf("create collection: %w", err)
		}

		// 创建索引
		idx, err := entity.NewIndexIvfFlat(entity.L2, 1024)
		if err != nil {
			return nil, fmt.Errorf("new index: %w", err)
		}

		err = cli.CreateIndex(ctx, cfg.Collection, "vector", idx, false)
		if err != nil {
			return nil, fmt.Errorf("create index: %w", err)
		}

		// 加载集合
		err = cli.LoadCollection(ctx, cfg.Collection, false)
		if err != nil {
			return nil, fmt.Errorf("load collection: %w", err)
		}

		log.Printf("集合 %s 创建成功", cfg.Collection)
	}

	// 创建自定义的VectorConverter，确保使用FloatVector
	vectorConverter := func(ctx context.Context, vectors [][]float64) ([]entity.Vector, error) {
		result := make([]entity.Vector, len(vectors))
		for i, vec := range vectors {
			// 转换为float32
			float32Vec := make([]float32, len(vec))
			for j, v := range vec {
				float32Vec[j] = float32(v)
			}
			// 创建FloatVector
			result[i] = entity.FloatVector(float32Vec)
		}
		return result, nil
	}

	// 创建检索器
	r, err := milvus.NewRetriever(ctx, &milvus.RetrieverConfig{
		Client:          cli,
		Collection:      cfg.Collection,
		Embedding:       emb,             // 提供 embedding 模型
		VectorField:     "vector",        // 明确指定向量字段名
		VectorConverter: vectorConverter, // 使用自定义转换器确保FloatVector类型
		MetricType:      entity.L2,       // 明确指定度量类型
	})
	if err != nil {
		if strings.Contains(err.Error(), "collection not found") {
			return nil, fmt.Errorf("集合 %s 不存在或无法访问，请确保Milvus服务正常运行: %w", cfg.Collection, err)
		}
		return nil, err
	}
	return &MilvusRAG{ret: r, cli: cli, cfg: cfg}, nil
}

// SearchTopK 对多条查询同时检索，返回按分数降序的 DocChunk。TopK 为每条查询的返回上限。
func (m *MilvusRAG) SearchTopK(ctx context.Context, queries []string, topK int, emb embedding.Embedder) ([]DocChunk, error) {
	if m.ret == nil {
		return nil, fmt.Errorf("milvus retriever not initialized")
	}
	var out []DocChunk
	for _, q := range queries {
		docs, err := m.ret.Retrieve(ctx, q, retriever.WithTopK(topK), retriever.WithEmbedding(emb))
		if err != nil {
			return nil, err
		}
		for _, it := range docs {
			meta := map[string]string{}
			for k, v := range it.MetaData {
				meta[k] = fmt.Sprint(v)
			}
			out = append(out, DocChunk{
				ID:       it.ID,
				Score:    it.Score(),
				Content:  it.Content,
				Metadata: meta,
			})
		}
	}
	return out, nil
}

// ImportDocument 导入文档到向量数据库
func (m *MilvusRAG) ImportDocument(ctx context.Context, content, docType, source string, metadata map[string]string, emb embedding.Embedder) error {
	if m.ret == nil {
		return fmt.Errorf("milvus retriever not initialized")
	}

	// 生成文档ID（使用统一助手）
	docID := GenerateDocID(content, docType, source)

	// 生成向量
	vectors, err := emb.EmbedStrings(ctx, []string{content})
	if err != nil {
		return fmt.Errorf("generate embedding: %w", err)
	}

	if len(vectors) == 0 {
		return fmt.Errorf("no vectors generated")
	}

	// 构建统一元数据
	metadata = BuildMetadata(docType, source, metadata)

	// 转换向量类型从float64到float32
	vector32 := make([]float32, len(vectors[0]))
	for i, v := range vectors[0] {
		vector32[i] = float32(v)
	}

	// 准备插入数据
	idColumn := entity.NewColumnVarChar("id", []string{docID})
	contentColumn := entity.NewColumnVarChar("content", []string{content})
	vectorColumn := entity.NewColumnFloatVector("vector", len(vectors[0]), [][]float32{vector32})
	metadataColumn := entity.NewColumnJSONBytes("metadata", [][]byte{mustMarshalJSON(metadata)})

	// 插入数据
	_, err = m.cli.Insert(ctx, m.cfg.Collection, "", idColumn, contentColumn, vectorColumn, metadataColumn)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}

	log.Printf("成功导入文档: %s (类型: %s, 来源: %s)", docID, docType, source)
	return nil
}

// mustMarshalJSON 辅助函数，将map转换为JSON字节数组
func mustMarshalJSON(data map[string]string) []byte {
	result := "{"
	first := true
	for k, v := range data {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":"%s"`, k, v)
		first = false
	}
	result += "}"
	return []byte(result)
}
