### Parameter List

| Parameter Name    | Type     | Required/Optional | Description                              |
|-------------------|----------|-------------------|------------------------------------------|
| `EMBEDDING_TYPE`  | `String` | Required          | embedding split api: openai gemini ernie huggingface |
| `EMBEDDING_BASE_URL` | `String` | Required for `huggingface` | Hugging Face TEI base URL such as `http://hf-embeddings:80` |
| `EMBEDDING_MODEL_ID` | `String` | Optional | embedding model id such as `BAAI/bge-small-zh-v1.5` |
| `EMBEDDING_QUERY_INSTRUCTION` | `String` | Optional | query-only prefix added before embedding lookup text |
| `KNOWLEDGE_PATH`  | `String` | Required          | knowledge doc path                       |
| `CHUNK_SIZE`      | `String` | Optional          | knowledge document chunk size            |
| `CHUNK_OVERLAP`   | `String` | Optional          | knowledge document chunk overlap         |
| `POSTGRES_DSN`    | `String` | Recommended       | PostgreSQL DSN for the unified knowledge store |
| `MINIO_ENDPOINT`  | `String` | Recommended       | MinIO endpoint for knowledge file storage |
| `REDIS_ADDR`      | `String` | Optional          | async ingestion queue address; without it ingestion is synchronous |
| `DEFAULT_KNOWLEDGE_BASE` | `String` | Optional | default knowledge base name |
| `DEFAULT_COLLECTION` | `String` | Optional | default collection name |
| `RERANKER_BASE_URL` | `String` | Optional | optional reranker endpoint |

Notes:

- The project now keeps a single default knowledge path and no longer requires the deprecated vector-backend environment variables.
- Milvus-related containers remain available only as an optional full profile, not as the default runtime or retrieval path.
