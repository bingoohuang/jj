# `wal`

A simple and fast write ahead log for Go.

## Features

- High durability
- Fast operations
- Monotonic indexes
- Batch writes
- Log truncation from front or back.

## Getting Started

### Example

```go
// open a new log file
wal, err := jj.WalOpen("mylog", nil)

// write some entries
err = wal.Write(1, []byte("first entry"))
err = wal.Write(2, []byte("second entry"))
err = wal.Write(3, []byte("third entry"))

// read an entry
data, err := wal.Read(1)
println(string(data)) // output: first entry

// close the log
err = wal.Close()
```

Batch writes:

```go

// write three entries as a batch
batch := new(jj.Batch)
batch.Write(1, []byte("first entry"))
batch.Write(2, []byte("second entry"))
batch.Write(3, []byte("third entry"))

err = wal.WriteBatch(batch)
```

Truncating:

```go
// write some entries
err = wal.Write(1, []byte("first entry"))
...
err = wal.Write(1000, []byte("thousandth entry"))

// truncate the log from index starting 350 and ending with 950.
err = l.TruncateFront(350)
err = l.TruncateBack(950)
```
