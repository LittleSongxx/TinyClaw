### 参数列表

| 参数名称             | 类型    | 是否必填 | 描述                           |
|------------------|-------|------|------------------------------|
| `EMBEDDING_TYPE` | `字符串` | 必填   | 向量化方式，支持：openai、gemini、ernie、huggingface |
| `EMBEDDING_BASE_URL` | `字符串` | `huggingface` 时必填 | Hugging Face TEI 服务地址，例如：`http://hf-embeddings:80` |
| `EMBEDDING_MODEL_ID` | `字符串` | 可选   | Embedding 模型 ID，例如：`BAAI/bge-small-zh-v1.5` |
| `EMBEDDING_QUERY_INSTRUCTION` | `字符串` | 可选   | 仅在 query 向量化时自动追加的检索前缀 |
| `KNOWLEDGE_PATH` | `字符串` | 必填   | 知识文档路径                       |
| `VECTOR_DB_TYPE` | `字符串` | 可选   | 向量数据库类型，例如：milvus,weaviate   |
| `CHROMA_URL`     | `字符串` | 可选   | Chroma 数据库的连接地址              |
| `MILVUS_URL`     | `字符串` | 可选   | Milvus 连接地址，例如：`milvus:19530` |
| `WEAVIATE_URL`   | `字符串` | 可选   | Weaviate 地址                     |
| `WEAVIATE_SCHEME`| `字符串` | 可选   | Weaviate 协议，例如：`http`        |
| `SPACE`          | `字符串` | 可选   | 向量数据库的命名空间（space name）       |
| `CHUNK_SIZE`     | `字符串` | 可选   | RAG 文件的切片大小                  |
| `CHUNK_OVERLAP`  | `字符串` | 可选   | RAG 文件的切片重叠大小                |
