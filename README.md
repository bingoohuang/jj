# jj

## jj.Pretty

jj.Pretty provides [fast](#performance) methods to format JSON for human readability or to uglify it for tiny payloads.

### Installing

To start using Pretty, install Go and run `go get`: `go get github.com/bingoohuang/jj`

### Pretty

Using this example:

```json
{"name":  {"first":"Tom","last":"Anderson"},  "age":37,
"children": ["Sara","Alex","Jack"],
"fav.movie": "Deer Hunter", "friends": [
    {"first": "Janet", "last": "Murphy", "age": 44}
  ]}
```

The following code:
```go
result = jj.Pretty(example)
```

Will format the json to:

```json
{
  "name": {
    "first": "Tom",
    "last": "Anderson"
  },
  "age": 37,
  "children": ["Sara", "Alex", "Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
    {
      "first": "Janet",
      "last": "Murphy",
      "age": 44
    }
  ]
}
```

### Color

Color will colorize the json for outputing to the screen. 

```go
result = jj.Color(json, nil)
```

Will add color to the result for printing to the terminal.
The second param is used for a customizing the style, and passing nil will use the default `pretty.TerminalStyle`.

### Ugly

The following code:

```go
result = jj.Ugly(example)
```

Will format the json to:

```json
{"name":{"first":"Tom","last":"Anderson"},"age":37,"children":["Sara","Alex","Jack"],"fav.movie":"Deer Hunter","friends":[{"first":"Janet","last":"Murphy","age":44}]}
```


### Customized output

There's a `PrettyOptions(json, opts)` function which allows for customizing the output with the following options:

```go
type Options struct {
	// Width is an max column width for single line arrays
	// Default 80
	Width int
	// Prefix is a prefix for all lines
	// Default empty
	Prefix string
	// Indent is the nested indentation
	// Default two spaces
	Indent string
	// SortKeys will sort the keys alphabetically
	// Default false
	SortKeys bool
}
```

### Performance

Benchmarks of Pretty alongside the builtin `encoding/json` Indent/Compact methods.

```sh
BenchmarkPretty-12               1000000              1113 ns/op             720 B/op          2 allocs/op
BenchmarkPrettySortKeys-12        562748              2149 ns/op            2848 B/op         14 allocs/op
BenchmarkUgly-12                 4303668               282 ns/op             240 B/op          1 allocs/op
BenchmarkUglyInPlace-12          5886506               203 ns/op               0 B/op          0 allocs/op
BenchmarkJSONIndent-12            430867              3262 ns/op            1277 B/op          0 allocs/op
BenchmarkJSONCompact-12           648189              1888 ns/op             467 B/op          0 allocs/op
```

*These benchmarks were run on a MacBook Pro 15" 2.2 GHz Intel Core i7 using Go 1.15.6
