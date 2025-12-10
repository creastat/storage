# Storage Implementation

## Overview

This directory contains implementations for various data storage solutions integrated into the Library Pipeline framework. Supports multiple backends for flexible data persistence and retrieval across different use cases.

## Supported Storage Providers

- **Redis**: High-performance key-value store for session management and caching
- **Supabase**: PostgreSQL-based solution with real-time capabilities
- **Qdrant**: Specialized vector database for embeddings storage
- **In-Memory**: Lightweight option for development and testing

## Configuration

Configure storage providers via environment variables or code:

```go
storageConfig := &pipeline.StorageConfig{
    Type:        "redis",
    Connection: "redis://user:pass@host:6379/0",
}
```

## Session Storage

Detailed implementation in [session/README.md](session/README.md)

## Vector Store

See [vectorstore/README.md](vectorstore/README.md) for Qdrant integration details

## Usage Example

```go
p := pipeline.NewPipelineWithConfig(storageConfig)
```

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for contribution guidelines

## License

[MIT License](../../LICENSE)
