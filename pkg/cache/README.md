# Generic Memory Cache

This package provides a generic, thread-safe in-memory cache implementation with support for:
- **Direct Access**: $O(1)$ key-value storage.
- **Secondary Indexing**: Relational-style querying via custom index extractors.
- **Batch Loading**: Efficient bulk import of data.

## 1. Architecture

The `MemoryCache` serves as a high-performance local storage component. It is built upon standard Go maps protected by `sync.RWMutex` for concurrent access.

```mermaid
graph TD
    User[Client Code] -->|Set/Get| Cache[MemoryCache]
    User -->|Find| Index[Index Engine]
    
    subgraph "pkg/cache"
        Cache -->|Primary Key| DataMap[map K V]
        Cache -->|Update| Index
        Index -->|Secondary Key| IndexMap[Index Storage]
        IndexMap -->|Lookup| DataMap
    end
```

## 2. Design

The design leverages Go Generics (`[K comparable, V any]`) to provide type safety without reflection overhead for standard operations. The `Store` interface extends the basic `Cache` capability with querying features.

```mermaid
classDiagram
    class Cache~K, V~ {
        <<interface>>
        +Set(key K, value V)
        +Get(key K) (V, bool)
        +Del(key K)
        +Load(items []V, keyFunc)
    }

    class Store~K, V~ {
        <<interface>>
        +AddIndex(name, extractor)
        +Find(indexName, val) ([]V, error)
        +Filter(predicate) []V
    }

    class MemoryCache~K, V~ {
        -mu sync.RWMutex
        -data map[K]V
        -extractors map[string]func(V)any
        -indices map[string]map[any]map[K]struct
        +NewMemoryCache()
    }

    Cache <|-- Store
    Store <|.. MemoryCache
```

### Internal Data Structures
- **Data**: `map[K]V` - Stores the actual objects.
- **Indices**: `map[indexName]map[indexValue]Set[K]` - Reverse lookup maps.
    - Level 1: Index Name (e.g., "role")
    - Level 2: Index Value (e.g., "admin")
    - Level 3: Set of Primary Keys (e.g., {1, 5, 8})

## 3. Workflow

### 3.1 Data Insertion & Indexing Flow

When data is inserted (`Set`) or loaded (`Load`), the cache automatically maintains all registered secondary indexes.

```mermaid
sequenceDiagram
    participant Client
    participant Cache as MemoryCache
    participant Index as Indexer
    participant Data as Data Map

    Client->>Cache: Set(Key, Value)
    activate Cache
    Cache->>Cache: Lock()
    
    rect rgb(200, 150, 150)
        Note over Cache, Data: Clean up Old Index
        Cache->>Data: Check existing (OldValue)
        opt Exists
            Cache->>Index: removeIndexEntry(OldValue)
        end
    end

    Cache->>Data: Store (Key, Value)

    rect rgb(150, 200, 150)
        Note over Cache, Index: Update New Index
        loop For Each Index
            Cache->>Index: Extract Value (extractor(Value))
            Cache->>Index: Add Entry (IndexName, IndexValue, Key)
        end
    end

    Cache->>Cache: Unlock()
    deactivate Cache
```

### 3.2 Query Flow

Querying via `Find` leverages the index for $O(1)$ complexity (amortized) relative to the number of items in the index bucket, avoiding full table scans.

```mermaid
sequenceDiagram
    participant Client
    participant Cache
    
    Client->>Cache: Find("role", "admin")
    activate Cache
    Cache->>Cache: RLock()
    
    Cache->>Cache: Lookup Index("role")
    Cache->>Cache: Lookup Bucket("admin") -> [Key1, Key2]
    
    loop For Each Key
        Cache->>Cache: Get Value from Data Map
    end
    
    Cache->>Cache: RUnlock()
    Cache-->>Client: Return []Value
    deactivate Cache
```
