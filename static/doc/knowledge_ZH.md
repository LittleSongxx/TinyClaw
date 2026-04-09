### 参数列表

| 参数名称             | 类型    | 是否必填 | 描述                           |
|------------------|-------|------|------------------------------|
| `EMBEDDING_TYPE` | `字符串` | 必填   | 向量化方式，支持：openai、gemini、ernie、huggingface |
| `EMBEDDING_BASE_URL` | `字符串` | `huggingface` 时必填 | Hugging Face TEI 服务地址，例如：`http://hf-embeddings:80` |
| `EMBEDDING_MODEL_ID` | `字符串` | 可选   | Embedding 模型 ID，例如：`BAAI/bge-small-zh-v1.5` |
| `EMBEDDING_QUERY_INSTRUCTION` | `字符串` | 可选   | 仅在 query 向量化时自动追加的检索前缀 |
| `KNOWLEDGE_PATH` | `字符串` | 必填   | 知识文档路径                       |
| `CHUNK_SIZE`     | `字符串` | 可选   | 知识文档的切片大小                  |
| `CHUNK_OVERLAP`  | `字符串` | 可选   | 知识文档的切片重叠大小                |
| `POSTGRES_DSN`   | `字符串` | 推荐   | 统一 knowledge 主存储 PostgreSQL 连接串 |
| `MINIO_ENDPOINT` | `字符串` | 推荐   | 统一 knowledge 文件存储 MinIO 地址    |
| `REDIS_ADDR`     | `字符串` | 可选   | 异步 ingestion 队列地址；不配则同步入库 |
| `DEFAULT_KNOWLEDGE_BASE` | `字符串` | 可选 | 默认知识库名 |
| `DEFAULT_COLLECTION` | `字符串` | 可选 | 默认 collection 名 |
| `RERANKER_BASE_URL` | `字符串` | 可选 | 可选 reranker 地址 |

说明：

- 当前项目默认只保留统一 knowledge 路径，不再要求配置旧的向量库环境变量。
- `Milvus` 相关容器仅作为可选 full profile 保留，不是默认启动和默认检索路径。
