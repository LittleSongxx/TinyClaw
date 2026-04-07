### Parameter List

| Parameter Name    | Type     | Required/Optional | Description                              |
|-------------------|----------|-------------------|------------------------------------------|
| `EMBEDDING_TYPE`  | `String` | Required          | embedding split api: openai gemini ernie huggingface |
| `EMBEDDING_BASE_URL` | `String` | Required for `huggingface` | Hugging Face TEI base URL such as `http://hf-embeddings:80` |
| `EMBEDDING_MODEL_ID` | `String` | Optional | embedding model id such as `BAAI/bge-small-zh-v1.5` |
| `EMBEDDING_QUERY_INSTRUCTION` | `String` | Optional | query-only prefix added before embedding lookup text |
| `KNOWLEDGE_PATH`  | `String` | Required          | knowledge doc path                       |
| `VECTOR_DB_TYPE`  | `String` | Required          | vector db type: weaviate milvus          |
| `CHROMA_URL`      | `String` | Optional          | chroma url:http://localhost:8080         |
| `MILVUS_URL`      | `String` | Optional          | milvus url: localhost:19530              |
| `WEAVIATE_URL`    | `String` | Optional          | weaviate url: localhost:8000             |
| `WEAVIATE_SCHEME` | `String` | Optional          | weaviate scheme: http                    |
| `SPACE`           | `String` | Optional          | vector db space name                     |
| `CHUNK_SIZE`      | `String` | Optional          | rag file chunk size                      |
| `CHUNK_OVERLAP`   | `String` | Optional          | rag file chunk overlap                   |
