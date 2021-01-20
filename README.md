# jj

JJ (get/set json values quickly) provides a [fast](#performance) and [simple](#get-a-value) way to get/set values from a json document.
It has features such as [one line retrieval](#get-a-value), [dot notation paths](#path-syntax), [iteration](#iterate-through-an-object-or-array), 
and [parsing json lines](#json-lines).

To start using jj, install Go and run `go get`: `go get github.com/bingoohuang/jj`

## Get/Set a value

jj.Get searches json for the specified path. A path is in dot syntax, such as "name.last" or "age". 
When the value is found it's returned immediately.

```go
package main

import (
	"fmt"
	"github.com/bingoohuang/jj"
)

func main() {
	jso := `{"name":{"first":"Janet","last":"Prichard"},"age":47}`
	value := jj.Get(jso, "name.last")
	fmt.Println(value.String()) // Print: Prichard
	jso, _ = jj.Set(jso, "name.last", "Anderson")
	fmt.Println(jso) //  Print:  {"name":{"first":"Janet","last":"Anderson"},"age":47}
}
```

*There's also the [jj.GetMany](#get-multiple-values-at-once) function to get multiple values at once, 
and [jj.GetBytes](#working-with-bytes) for working with JSON byte slices.*

## Get Path Syntax

Below is a quick overview of the path syntax, for more complete information please
check out [GJSON Syntax](SYNTAX.md).

- A path is a series of keys separated by a dot.
- A key may contain special wildcard characters '\*' and '?'.
- To access an array value use the index as the key.
- To get the number of elements in an array or to access a child path, use the '#' character.
- The dot and wildcard characters can be escaped with '\\'.

```json
{
  "name": {"first": "Tom", "last": "Anderson"},
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
    {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
    {"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
    {"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
  ]
}
```

```
"name.last"          >> "Anderson"
"age"                >> 37
"children"           >> ["Sara","Alex","Jack"]
"children.#"         >> 3
"children.1"         >> "Alex"
"child*.2"           >> "Jack"
"c?ildren.0"         >> "Sara"
"fav\.movie"         >> "Deer Hunter"
"friends.#.first"    >> ["Dale","Roger","Jane"]
"friends.1.last"     >> "Craig"
```

You can also query an array for the first match by using `#(...)`, or find all
matches with `#(...)#`. Queries support the `==`, `!=`, `<`, `<=`, `>`, `>=`
comparison operators and the simple pattern matching `%` (like) and `!%`
(not like) operators.

```
friends.#(last=="Murphy").first    >> "Dale"
friends.#(last=="Murphy")#.first   >> ["Dale","Jane"]
friends.#(age>45)#.last            >> ["Craig","Murphy"]
friends.#(first%"D*").last         >> "Murphy"
friends.#(first!%"D*").last        >> "Craig"
friends.#(nets.#(=="fb"))#.first   >> ["Dale","Roger"]
```

## Set Path syntax

A path is a series of keys separated by a dot.
The dot and colon characters can be escaped with ``\``.

```json
{
  "name": {"first": "Tom", "last": "Anderson"},
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
	{"first": "James", "last": "Murphy"},
	{"first": "Roger", "last": "Craig"}
  ]
}
```
```
"name.last"          >> "Anderson"
"age"                >> 37
"children.1"         >> "Alex"
"friends.1.last"     >> "Craig"
"children.-1"        >> appends a new value to the end of the children array
```

Normally number keys are used to modify arrays, but it's possible to force a numeric object key by using the colon character:

```json
{
  "users":{
    "2313":{"name":"Sara"},
    "7839":{"name":"Andy"}
  }
}
```

A colon path would look like:

```
"users.:2313.name"    >> "Sara"
```

Supported types
---------------

Pretty much any type is supported:

```go
jj.Set(`{"key":true}`, "key", nil)
jj.Set(`{"key":true}`, "key", false)
jj.Set(`{"key":true}`, "key", 1)
jj.Set(`{"key":true}`, "key", 10.5)
jj.Set(`{"key":true}`, "key", "hello")
jj.Set(`{"key":true}`, "key", map[string]interface{}{"hello":"world"})
```

When a type is not recognized, SJSON will fallback to the `encoding/json` Marshaller.


Examples
--------

```go
// Set a value from empty document:
value, _ := jj.Set("", "name", "Tom")
println(value) // Output: {"name":"Tom"}

// Set a nested value from empty document:
value, _ = jj.Set("", "name.last", "Anderson")
println(value)  // Output: {"name":{"last":"Anderson"}}

// Set a new value:
value, _ = jj.Set(`{"name":{"last":"Anderson"}}`, "name.first", "Sara")
println(value) // Output: {"name":{"first":"Sara","last":"Anderson"}}

// Update an existing value:
value, _ = jj.Set(`{"name":{"last":"Anderson"}}`, "name.last", "Smith")
println(value) // Output: {"name":{"last":"Smith"}}

// Set a new array value:
value, _ = jj.Set(`{"friends":["Andy","Carol"]}`, "friends.2", "Sara")
println(value) // Output: {"friends":["Andy","Carol","Sara"]

// Append an array value by using the `-1` key in a path:
value, _ = jj.Set(`{"friends":["Andy","Carol"]}`, "friends.-1", "Sara")
println(value) // Output: {"friends":["Andy","Carol","Sara"]

// Append an array value that is past the end:
value, _ = jj.Set(`{"friends":["Andy","Carol"]}`, "friends.4", "Sara")
println(value) // Output: {"friends":["Andy","Carol",null,null,"Sara"]

// Delete a value:
value, _ = jj.Delete(`{"name":{"first":"Sara","last":"Anderson"}}`, "name.first")
println(value)  // Output: {"name":{"last":"Anderson"}}

// Delete an array value:
value, _ = jj.Delete(`{"friends":["Andy","Carol"]}`, "friends.1")
println(value) // Output: {"friends":["Andy"]}

// Delete the last array value:
value, _ = jj.Delete(`{"friends":["Andy","Carol"]}`, "friends.-1")
println(value) // Output: {"friends":["Andy"]}
```

## Result Type

jj.Get supports the json types `string`, `number`, `bool`, and `null`.
Arrays and Objects are returned as their raw json types.

The `Result` type holds one of these:

```
bool, for JSON booleans
float64, for JSON numbers
string, for JSON string literals
nil, for JSON null
```

To directly access the value:

```go
result.Type    // can be String, Number, True, False, Null, or JSON
result.Str     // holds the string
result.Num     // holds the float64 number
result.Raw     // holds the raw json
result.Index   // index of raw value in original json, zero means index unknown
```

There are a variety of handy functions that work on a result:

```go
result.Exists() bool
result.Value() interface{}
result.Int() int64
result.Uint() uint64
result.Float() float64
result.String() string
result.Bool() bool
result.Time() time.Time
result.Array() []jj.Result
result.Map() map[string]jj.Result
result.Get(path string) Result
result.ForEach(iterator func(key, value Result) bool)
result.Less(token Result, caseSensitive bool) bool
```

The `result.Value()` function returns an `interface{}` which requires type assertion and is one of the following Go types:

The `result.Array()` function returns an array of values.
If the result represents a non-existent value, then an empty array will be returned.
If the result is not a JSON array, the return value will be an array containing one result.

```go
boolean >> bool
number  >> float64
string  >> string
null    >> nil
array   >> []interface{}
object  >> map[string]interface{}
```

### 64-bit integers

The `result.Int()` and `result.Uint()` calls are capable of reading all 64 bits, allowing for large JSON integers.

```go
result.Int() int64    // -9223372036854775808 to 9223372036854775807
result.Uint() int64   // 0 to 18446744073709551615
```

## Modifiers and path chaining

A modifier is a path component that performs custom processing on the json.

Multiple paths can be "chained" together using the pipe character.
This is useful for getting results from a modified query.

For example, using the built-in `@reverse` modifier on the above json document,
we'll get `children` array and reverse the order:

```
"children|@reverse"           >> ["Jack","Alex","Sara"]
"children|@reverse|0"         >> "Jack"
```

There are currently the following built-in modifiers:

- `@reverse`: Reverse an array or the members of an object.
- `@ugly`: Remove all whitespace from a json document.
- `@pretty`: Make the json document more human readable.
- `@this`: Returns the current element. It can be used to retrieve the root element.
- `@valid`: Ensure the json document is valid.
- `@flatten`: Flattens an array.
- `@join`: Joins multiple objects into a single object.

### Modifier arguments

A modifier may accept an optional argument. The argument can be a valid JSON
document or just characters.

For example, the `@pretty` modifier takes a json object as its argument.

```
@pretty:{"sortKeys":true} 
```

Which makes the json pretty and orders all of its keys.

```json
{
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
    {"age": 44, "first": "Dale", "last": "Murphy"},
    {"age": 68, "first": "Roger", "last": "Craig"},
    {"age": 47, "first": "Jane", "last": "Murphy"}
  ],
  "name": {"first": "Tom", "last": "Anderson"}
}
```

*The full list of `@pretty` options are `sortKeys`, `indent`, `prefix`, and `width`.
Please see [Pretty Options](#customized-output) for more information.*

### Custom modifiers

You can also add custom modifiers.

For example, here we create a modifier that makes the entire json document upper
or lower case.

```go
jj.AddModifier("case", func(jso, arg string) string {
  if arg == "upper" {
    return strings.ToUpper(jso)
  }
  if arg == "lower" {
    return strings.ToLower(jso)
  }
  return json
})
```

```
"children|@case:upper"           >> ["SARA","ALEX","JACK"]
"children|@case:lower|@reverse"  >> ["jack","alex","sara"]
```

### JSON Lines

There's support for [JSON Lines](http://jsonlines.org/) using the `..` prefix, which treats a multilined document as an array.

For example:

```
{"name": "Gilbert", "age": 61}
{"name": "Alexa", "age": 34}
{"name": "May", "age": 57}
{"name": "Deloise", "age": 44}
```

```
..#                   >> 4
..1                   >> {"name": "Alexa", "age": 34}
..3                   >> {"name": "Deloise", "age": 44}
..#.name              >> ["Gilbert","Alexa","May","Deloise"]
..#(name="May").age   >> 57
```

The `ForEachLines` function will iterate through JSON lines.

```go
jj.ForEachLine(json, func(line jj.Result) bool{
    println(line.String())
    return true
})
```

## Get nested array values

Suppose you want all the last names from the following json:

```json
{
  "programmers": [
    {
      "firstName": "Janet", 
      "lastName": "McLaughlin", 
    }, {
      "firstName": "Elliotte", 
      "lastName": "Hunter", 
    }, {
      "firstName": "Jason", 
      "lastName": "Harold", 
    }
  ]
}
```

You would use the path "programmers.#.lastName" like such:

```go
result := jj.Get(json, "programmers.#.lastName")
for _, name := range result.Array() {
	println(name.String())
}
```

You can also query an object inside an array:

```go
name := jj.Get(json, `programmers.#(lastName="Hunter").firstName`)
println(name.String())  // prints "Elliotte"
```

## Iterate through an object or array

The `ForEach` function allows for quickly iterating through an object or array.
The key and value are passed to the iterator function for objects.
Only the value is passed for arrays.
Returning `false` from an iterator will stop iteration.

```go
result := jj.Get(json, "programmers")
result.ForEach(func(key, value jj.Result) bool {
	println(value.String()) 
	return true // keep iterating
})
```

## Simple Parse and Get

There's a `Parse(json)` function that will do a simple parse, and `result.Get(path)` that will search a result.

For example, all of these will return the same result:

```go
jj.Parse(json).Get("name").Get("last")
jj.Get(json, "name").Get("last")
jj.Get(json, "name.last")
```

## Check for the existence of a value

Sometimes you just want to know if a value exists.

```go
value := jj.Get(json, "name.last")
if !value.Exists() {
	println("no last name")
} else {
	println(value.String())
}

// Or as one step
if jj.Get(json, "name.last").Exists() {
	println("has a last name")
}
```

## Validate JSON

The `Get*` and `Parse*` functions expects that the json is well-formed. Bad json will not panic, but it may return back unexpected results.

If you are consuming JSON from an unpredictable source then you may want to validate prior to using GJSON.

```go
if !jj.Valid(json) {
	return errors.New("invalid json")
}
value := jj.Get(json, "name.last")
```

## Unmarshal to a map

To unmarshal to a `map[string]interface{}`:

```go
m, ok := jj.Parse(json).Value().(map[string]interface{})
if !ok {
	// not a map
}
```

## Working with Bytes

If your JSON is contained in a `[]byte` slice, there's the GetBytes function. This is preferred over `Get(string(data), path)`.

```go
var json []byte = ...
result := jj.GetBytes(json, path)
```

If you are using the `jj.GetBytes(json, path)` function and you want to avoid converting `result.Raw` to a `[]byte`, then you can use this pattern:

```go
var json []byte = ...
result := jj.GetBytes(json, path)
var raw []byte
if result.Index > 0 {
    raw = json[result.Index:result.Index+len(result.Raw)]
} else {
    raw = []byte(result.Raw)
}
```

This is a best-effort no allocation sub slice of the original json. This method utilizes the `result.Index` field, which is the position of the raw data in the original json. It's possible that the value of `result.Index` equals zero, in which case the `result.Raw` is converted to a `[]byte`.

## Get multiple values at once

The `GetMany` function can be used to get multiple values at the same time.

```go
results := jj.GetMany(json, "name.first", "name.last", "age")
```

The return value is a `[]Result`, which will always contain exactly the same number of items as the input paths.

## Performance

Benchmarks of jj with

- [encoding/json](https://golang.org/pkg/encoding/json/),
- [ffjson](https://github.com/pquerna/ffjson),
- [EasyJSON](https://github.com/mailru/easyjson),
- [jsonparser](https://github.com/buger/jsonparser),
- [json-iterator](https://github.com/json-iterator/go)

```
BenchmarkGJSONGet-12                     3000000               445 ns/op               1 B/op          1 allocs/op
BenchmarkGJSONUnmarshalMap-12             872274              4182 ns/op            1920 B/op         26 allocs/op
BenchmarkJSONUnmarshalMap-12              421632              8727 ns/op            2984 B/op         69 allocs/op
BenchmarkJSONUnmarshalStruct-12           579792              5510 ns/op             912 B/op         12 allocs/op
BenchmarkJSONDecoder-12                   303375             13905 ns/op            4026 B/op        160 allocs/op
BenchmarkFFJSONLexer-12                  1038411              3532 ns/op             896 B/op          8 allocs/op
BenchmarkEasyJSONLexer-12                3000000              1008 ns/op             501 B/op          5 allocs/op
BenchmarkJSONParserGet-12                3000000               560 ns/op              21 B/op          0 allocs/op
BenchmarkJSONIterator-12                 3000000              1051 ns/op             693 B/op         14 allocs/op
```

JSON document used:

```json
{
  "widget": {
    "debug": "on",
    "window": {
      "title": "Sample Konfabulator Widget",
      "name": "main_window",
      "width": 500,
      "height": 500
    },
    "image": { 
      "src": "Images/Sun.png",
      "hOffset": 250,
      "vOffset": 250,
      "alignment": "center"
    },
    "text": {
      "data": "Click Here",
      "size": 36,
      "style": "bold",
      "vOffset": 100,
      "alignment": "center",
      "onMouseUp": "sun1.opacity = (sun1.opacity / 100) * 90;"
    }
  }
}    
```

Each operation was rotated through one of the following search paths:

```
widget.window.name
widget.image.hOffset
widget.text.onMouseUp
```

*These benchmarks were run on a MacBook Pro 15" 2.8 GHz Intel Core i7 using Go 1.8 and can be found [BENCH.md](BENCH.md).*

## jj.Match

jj.Match is a very simple pattern matcher where '*' matches on any number characters and '?' matches on any one
character.

## Example

```go
jj.Match("hello", "*llo")
jj.Match("jello", "?ello")
jj.Match("hello", "h*o") 
```

## jj.Pretty

jj.Pretty provides [fast](#performance) methods to format JSON for human readability or to uglify it for tiny payloads.

### Pretty

Using this example:

```json
{
  "name": {
    "first": "Tom",
    "last": "Anderson"
  },
  "age": 37,
  "children": [
    "Sara",
    "Alex",
    "Jack"
  ],
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
  "children": [
    "Sara",
    "Alex",
    "Jack"
  ],
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

Will add color to the result for printing to the terminal. The second param is used for a customizing the style, and
passing nil will use the default `pretty.TerminalStyle`.

### Ugly

The following code:

```go
result = jj.Ugly(example)
```

Will format the json to:

```json
{
  "name": {
    "first": "Tom",
    "last": "Anderson"
  },
  "age": 37,
  "children": [
    "Sara",
    "Alex",
    "Jack"
  ],
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

## jj tools

jj is another JSON Stream Editor.

jj is a command line utility that provides a [fast](#performance) and simple way to retrieve or update values from JSON documents.

It's [fast](#performance) because it avoids parsing irrelevant sections of json, 
skipping over values that do not apply, and aborts as soon as the target value has been found or updated.

Install `go get github.com/bingoohuang/jj/...`

### Usage

```
$ jj -h

usage: jj [-v value] [-urOD] [-i infile] [-o outfile] keypath

examples: jj keypath                      read value from stdin
      or: jj -i infile keypath            read value from infile
      or: jj -v value keypath             edit value
      or: jj -v value -o outfile keypath  edit value and write to outfile

options:
      -v value             Edit JSON key path value
      -u                   Make json ugly, keypath is optional
      -r                   Use raw values, otherwise types are auto-detected
      -n                   Do not output color or extra formatting
      -O                   Performance boost for value updates
      -D                   Delete the value at the specified key path
      -l                   Output array values on multiple lines
      -i infile            Use input file instead of stdin
      -o outfile           Use output file instead of stdout
      keypath              JSON key path (like "name.last")
```

### Examples

#### Getting a value

jj uses a [path syntax](SYNTAX.md) for finding values.

Get a string:
```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj name.last
Smith
```

Get a block of JSON:
```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj name
{"first":"Tom","last":"Smith"}
```

Try to get a non-existent key:
```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj name.middle
null
```

Get the raw string value:
```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj -r name.last
"Smith"
```

Get an array value by index:
```sh
$ echo '{"friends":["Tom","Jane","Carol"]}' | jj friends.1
Jane
```

#### JSON Lines

There's support for [JSON Lines](http://jsonlines.org/) using the `..` path prefix.
Which when specified, treats the multi-lined document as an array.

For example:

```
{"name": "Gilbert", "age": 61}
{"name": "Alexa", "age": 34}
{"name": "May", "age": 57}
```

```
..#                   >> 3
..1                   >> {"name": "Alexa", "age": 34}
..#.name              >> ["Gilbert","Alexa","May"]
..#[name="May"].age   >> 57
```

```sh
$ jj -i testdata/line.json  ..#
3
$ jj -i testdata/line.json  ..1   
{"name": "Alexa", "age": 34}
$ jj -i testdata/line.json  ..#.name   
["Gilbert","Alexa","May"]
$ jj -i testdata/line.json  "..#[name="May"].age"
57
```

#### Setting a value

The [path syntax](#set-path-syntax) for setting values has a couple of tiny differences than for getting values.

The `-v value` option is auto-detected as a Number, Boolean, Null, or String.
You can override the auto-detection and input raw JSON by including the `-r` option.
This is useful for raw JSON blocks such as object, arrays, or premarshalled strings.

Update a value:

```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj -v Andy name.first
{"name":{"first":"Andy","last":"Smith"}}
```

Set a new value:

```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj -v 46 age
{"age":46,"name":{"first":"Tom","last":"Smith"}}
```

Set a new nested value:

```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj -v relax task.today
{"task":{"today":"relax"},"name":{"first":"Tom","last":"Smith"}}
```

Replace an array value by index:

```sh
$ echo '{"friends":["Tom","Jane","Carol"]}' | jj -v Andy friends.1
{"friends":["Tom","Andy","Carol"]}
```

Append an array:

```sh
$ echo '{"friends":["Tom","Jane","Carol"]}' | jj -v Andy friends.-1
{"friends":["Tom","Andy","Carol","Andy"]}
```

Set an array value that's past the bounds:

```sh
$ echo '{"friends":["Tom","Jane","Carol"]}' | jj -v Andy friends.5
{"friends":["Tom","Andy","Carol",null,null,"Andy"]}
```

Set a raw block of JSON:

```sh
$ echo '{"name":"Carol"}' | jj -r -v '["Tom","Andy"]' friends
{"friends":["Tom","Andy"],"name":"Carol"}
```

Start new JSON document:

```sh
$ echo '' | jj -v 'Sam' name.first
{"name":{"first":"Sam"}}
```

#### Deleting a value

Delete a value:

```sh
$ echo '{"age":46,"name":{"first":"Tom","last":"Smith"}}' | jj -D age
{"name":{"first":"Tom","last":"Smith"}}
```

Delete an array value by index:

```sh
$ echo '{"friends":["Andy","Carol"]}' | jj -D friends.0
{"friends":["Carol"]}
```

Delete last item in array:

```sh
$ echo '{"friends":["Andy","Carol"]}' | jj -D friends.-1
{"friends":["Andy"]}
```

#### Optimistically update a value

The `-O` option can be used when the caller expects that a value at the
specified keypath already exists.

Using this option can speed up an operation by as much as 6x, but
slow down as much as 20% when the value does not exist.

For example:

```sh
echo '{"name":{"first":"Tom","last":"Smith"}}' | jj -v Tim -O name.first
```

The `-O` tells jj that the `name.first` likely exists so try a fasttrack operation first.

#### Pretty printing

```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj name
{
  "first": "Tom",
  "last": "Smith"
}
```

Also the keypath is optional, allowing for the entire json document to be made pretty.

```sh
$ echo '{"name":{"first":"Tom","last":"Smith"}}' | jj
{
  "name": {
    "first": "Tom",
    "last": "Smith"
  }
}
```

#### Ugly printing

The `-u` flag will compress the json into the fewest characters possible by squashing newlines and spaces.

#### Performance

A quick comparison of jj to [jq](https://stedolan.github.io/jq/). 
The test [json file](https://github.com/tidwall/sf-city-lots-json) is 180MB file of 206,560 city parcels in San Francisco.

*Tested on a 2018 Macbook Pro running jq 1.6 and jj 1.0.0*

#### Get a lot of number for the parcel at index 10000

```sh
$ time cat citylots.json | jq -cM ".features[10000].properties.LOT_NUM"
"091"
cat citylots.json  0.01s user 0.11s system 2% cpu 5.010 total
jq -cM ".features[10000].properties.LOT_NUM"  5.46s user 0.66s system 99% cpu 6.151 total

$ time cat citylots.json | jj -r features.10000.properties.LOT_NUM
"091"
cat citylots.json  0.01s user 0.10s system 24% cpu 0.449 total
jj -r features.10000.properties.LOT_NUM  0.24s user 0.28s system 107% cpu 0.494 total
```

#### Update the lot number for the parcel at index 10000

```sh
$ time cat citylots.json | jq -cM '.features[10000].properties.LOT_NUM="12A"' > /dev/null
cat citylots.json  0.01s user 0.08s system 1% cpu 5.452 total
jq -cM '.features[10000].properties.LOT_NUM="12A"' > /dev/null  13.94s user 0.74s system 99% cpu 14.772 total

$ time cat citylots.json | jj -O -v 12A features.10000.properties.LOT_NUM > /dev/null
cat citylots.json  0.01s user 0.08s system 23% cpu 0.368 total
jj -O -v 12A features.10000.properties.LOT_NUM > /dev/null  0.22s user 0.27s system 121% cpu 0.406 total
```
