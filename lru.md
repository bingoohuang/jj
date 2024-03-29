# `tinylru`

A fast little LRU cache.

## Getting Started

### Usage

```go
// Create an LRU cache
var cache jj.LRU

// Set the cache size. This is the maximum number of items that the cache can
// hold before evicting old items. The default size is 256.
cache.Resize(1024)

// Set a key. Returns the previous value and ok if a previous value exists.
prev, ok := cache.Set("hello", "world")

// Get a key. Returns the value and ok if the value exists.
value, ok := cache.Get("hello")

// Delete a key. Returns the deleted value and ok if a previous value exists.
prev, ok := tr.Delete("hello")
```

A `Set` function may evict old items when adding a new item while LRU is at capacity. If you want to know what was
evicted then use the `SetEvicted`
function.

```go
// Set a key and return the evicted item, if any.
prev, ok, evictedKey, evictedValue, evicted := cache.SetEvicted("hello", "jello")
```

 