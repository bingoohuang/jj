package jj_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bingoohuang/jj"
)

func ExampleStreamParse() {
	const JSON = `
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
	`
	// A JSON stream parser
	jj.StreamParse([]byte(JSON), func(start, end, info int) int {
		if jj.IsToken(info, jj.TokString) || jj.IsToken(info, jj.TokValue) {
			fmt.Println(JSON[start:end])
		}
		return 1
	})

	// OpsOutput:
	// "Tom"
	// "Anderson"
	// "Sara"
	// "Alex"
	// "Jack"
	// "Deer Hunter"
	// "Dale"
	// "Murphy"
	// "ig"
	// "fb"
	// "tw"
	// "Roger"
	// "Craig"
	// "fb"
	// "tw"
	// "Jane"
	// "Murphy"
	// "ig"
	// "tw"
}

var json1 = `{
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
}`

var json2 = `
{
	"tagged": "OK",
	"Tagged": "KO",
	"NotTagged": true,
	"unsettable": 101,
	"Nested": {
		"Yellow": "Green",
		"yellow": "yellow"
	},
	"nestedTagged": {
		"Green": "Green",
		"Map": {
			"this": "that",
			"and": "the other thing"
		},
		"Ints": {
			"Uint": 99,
			"Uint16": 16,
			"Uint32": 32,
			"Uint64": 65
		},
		"Uints": {
			"int": -99,
			"Int": -98,
			"Int16": -16,
			"Int32": -32,
			"int64": -64,
			"Int64": -65
		},
		"Uints": {
			"Float32": 32.32,
			"Float64": 64.64
		},
		"Byte": 254,
		"Bool": true
	},
	"LeftOut": "you shouldn't be here",
	"SelfPtr": {"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}},
	"SelfSlice": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"SelfSlicePtr": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"SelfPtrSlice": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"interface": "Tile38 Rocks!",
	"Interface": "Please Download",
	"Array": [0,2,3,4,5],
	"time": "2017-05-07T13:24:43-07:00",
	"Binary": "R0lGODlhPQBEAPeo",
	"NonBinary": [9,3,100,115]
}
`

func mustEqual(a, b string) {
	if a != b {
		panic("'" + a + "' != '" + b + "'")
	}
}

func TestIter(t *testing.T) {
	var json []byte
	var out []byte

	json = []byte(` { "hello" : [ 1, 2, 3 ], "jello" : [ 4, 5, 6 ] } `)
	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		return -1
	})
	mustEqual(string(out), "{}")

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		return 0
	})
	mustEqual(string(out), "{")

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		return -1
	})
	mustEqual(string(out), "{}")

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		if jj.IsToken(info, jj.TokKey) {
			return 0
		}
		return 1
	})
	mustEqual(string(out), `{"hello"`)

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		if jj.IsToken(info, jj.TokColon) {
			return 0
		}
		return 1
	})
	mustEqual(string(out), `{"hello":`)

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		if jj.IsToken(info, jj.TokOpen|jj.TokArray) {
			return -1
		}
		if jj.IsToken(info, jj.TokComma) {
			return 0
		}
		return 1
	})
	mustEqual(string(out), `{"hello":[],`)

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		if jj.IsToken(info, jj.TokOpen|jj.TokArray) {
			return -1
		}
		return 1
	})
	mustEqual(string(out), `{"hello":[],"jello":[]}`)

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		if jj.IsToken(info, jj.TokOpen|jj.TokArray) {
			return -1
		}
		if jj.IsToken(info, jj.TokClose|jj.TokObject) {
			return 0
		}
		return 1
	})
	mustEqual(string(out), `{"hello":[],"jello":[]}`)

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		if jj.IsToken(info, jj.TokObject|jj.TokStart) {
			out = append(out, json[start:end]...)
		}
		return 0
	})
	mustEqual(string(out), "{")

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		if jj.IsToken(info, jj.TokObject|jj.TokStart|jj.TokEnd) {
			out = append(out, json[start:end]...)
		}
		return 0
	})
	mustEqual(string(out), "")

	json = []byte(" [ 1,2,3 ] ")
	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		return 0
	})
	mustEqual(string(out), "[")

	json = []byte(" [ 1,2,3 ] ")
	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		if jj.IsToken(info, jj.TokComma) {
			return 0
		}
		return 1
	})
	mustEqual(string(out), "[1,")

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		return -1
	})
	mustEqual(string(out), "[]")

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		if jj.IsToken(info, jj.TokArray|jj.TokClose) {
			return 0
		}
		return 1
	})
	mustEqual(string(out), "[1,2,3]")

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		if jj.IsToken(info, jj.TokArray|jj.TokStart) {
			out = append(out, json[start:end]...)
		}
		return 0
	})
	mustEqual(string(out), "[")

	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		if jj.IsToken(info, jj.TokArray|jj.TokStart|jj.TokEnd) {
			out = append(out, json[start:end]...)
		}
		return 0
	})
	mustEqual(string(out), "")

	json = []byte(" true ")
	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		return 0
	})
	mustEqual(string(out), "true")

	json = []byte(" true ")
	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		if jj.IsToken(info, jj.TokStart|jj.TokEnd) {
			out = append(out, json[start:end]...)
			return 0
		}
		return 1
	})
	mustEqual(string(out), "true")

	json = []byte(`{  "hi\nthere": "yo" }`)
	out = nil
	jj.StreamParse(json, func(start, end, info int) int {
		if jj.IsToken(info, jj.TokKey) {
			out = append(out, json[start:end]...)
			return 0
		}
		return 1
	})
	mustEqual(string(out), `"hi\nthere"`)

	json = []byte(` { "a" : "b" , "c" : [ 1 , 2 , 3 ] } `)
	out = nil
	var index int
	expect := []int{
		jj.TokStart | jj.TokOpen | jj.TokObject,
		jj.TokKey | jj.TokString,
		jj.TokColon,
		jj.TokValue | jj.TokString,
		jj.TokComma,
		jj.TokKey | jj.TokString,
		jj.TokColon,
		jj.TokValue | jj.TokOpen | jj.TokArray,
		jj.TokValue | jj.TokNumber,
		jj.TokComma,
		jj.TokValue | jj.TokNumber,
		jj.TokComma,
		jj.TokValue | jj.TokNumber,
		jj.TokValue | jj.TokClose | jj.TokArray,
		jj.TokEnd | jj.TokClose | jj.TokObject,
	}
	jj.StreamParse(json, func(start, end, info int) int {
		if expect[index] != info {
			t.Fatalf("expected %d, got %d (#%d)\n", expect[index], info, index)
			return 0
		}
		index++
		return 1
	})
	if index != 15 {
		panic("!")
	}
	// mustEqual(string(out), "true")
}

func testreturnvalue(t *testing.T, json string, expect int) {
	t.Helper()
	e := jj.StreamParse([]byte(json), nil)
	if e != expect {
		t.Fatalf("expected '%d', got '%d'", expect, e)
	}
}

func TestReturnValues(t *testing.T) {
	testreturnvalue(t, "false", 5)
	testreturnvalue(t, "false ", 6)
	testreturnvalue(t, " false ", 7)
	testreturnvalue(t, "", 0)
	testreturnvalue(t, " ", -1)
	testreturnvalue(t, " a", -1)
	testreturnvalue(t, ` {"hel\y" : 1}`, -7)
}

func testvalid(t *testing.T, json string, expect bool) {
	t.Helper()
	e := jj.StreamParse([]byte(json), nil)
	ok := e > 0
	if ok != expect {
		t.Fatal("mismatch")
	}
}

func TestValidBasic(t *testing.T) {
	testvalid(t, "false", true)
	testvalid(t, "fals0", false)
	testvalid(t, "-\n", false)
	testvalid(t, "0", true)
	testvalid(t, "00", false)
	testvalid(t, "-00", false)
	testvalid(t, "-.", false)
	testvalid(t, "0.0", true)
	testvalid(t, "10.0", true)
	testvalid(t, "10e1", true)
	testvalid(t, "10EE", false)
	testvalid(t, "10E-", false)
	testvalid(t, "10E+", false)
	testvalid(t, "10E+1a", false)
	testvalid(t, "10E123", true)
	testvalid(t, "10E-123", true)
	testvalid(t, "10E-0123", true)
	testvalid(t, "", false)
	testvalid(t, " ", false)
	testvalid(t, "{}", true)
	testvalid(t, "{", false)
	testvalid(t, "-", false)
	testvalid(t, "-1", true)
	testvalid(t, "-1.", false)
	testvalid(t, "-1.0", true)
	testvalid(t, " -1.0", true)
	testvalid(t, " -1.0 ", true)
	testvalid(t, "-1.0 ", true)
	testvalid(t, "-1.0 i", false)
	testvalid(t, "-1.0 i", false)
	testvalid(t, "true", true)
	testvalid(t, " true", true)
	testvalid(t, " true ", true)
	testvalid(t, " True ", false)
	testvalid(t, " tru", false)
	testvalid(t, "false", true)
	testvalid(t, " false", true)
	testvalid(t, " false ", true)
	testvalid(t, " False ", false)
	testvalid(t, " fals", false)
	testvalid(t, "null", true)
	testvalid(t, " null", true)
	testvalid(t, " null ", true)
	testvalid(t, " Null ", false)
	testvalid(t, " nul", false)
	testvalid(t, " []", true)
	testvalid(t, " [true]", true)
	testvalid(t, " [ true, null ]", true)
	testvalid(t, " [ true,]", false)
	testvalid(t, `{"hello":"world"}`, true)
	testvalid(t, `{ "hello": "world" }`, true)
	testvalid(t, `{ "hello": "world", }`, false)
	testvalid(t, `{"a":"b",}`, false)
	testvalid(t, `{"a":"b","a"}`, false)
	testvalid(t, `{"a":"b","a":}`, false)
	testvalid(t, `{"a":"b","a":1}`, true)
	testvalid(t, `{"a":"b",2"1":2}`, false)
	testvalid(t, `{"a":"b","a": 1, "c":{"hi":"there"} }`, true)
	testvalid(t, `{"a":"b","a": 1, "c":{"hi":"there", "easy":["going",`+
		`{"mixed":"bag"}]} }`, true)
	testvalid(t, `""`, true)
	testvalid(t, `"`, false)
	testvalid(t, `"\n"`, true)
	testvalid(t, `"\"`, false)
	testvalid(t, `"\\"`, true)
	testvalid(t, `"a\\b"`, true)
	testvalid(t, `"a\\b\\\"a"`, true)
	testvalid(t, `"a\\b\\\uFFAAa"`, true)
	testvalid(t, `"a\\b\\\uFFAZa"`, false)
	testvalid(t, `"a\\b\\\uFFA"`, false)
	testvalid(t, string(json1), true)
	testvalid(t, string(json2), true)
	testvalid(t, `"hello`+string(byte(0))+`world"`, false)
	testvalid(t, `"hello world\`, false)
	testvalid(t, `"hello world\i`, false)
	testvalid(t, `"hello world\u8`, false)
	testvalid(t, `[1`, false)
	testvalid(t, `[1,`, false)
	testvalid(t, `{"hi":"ya"`, false)
	testvalid(t, `{"hi`, false)
	testvalid(t, `{123:123}`, false)
	testvalid(t, `123.a123`, false)
	testvalid(t, `123.123e`, false)
}

// mustBeGood parses JSON and stitches together a new JSON document and checks
// if the new doc matches the original.
func mustBeAGood(json []byte) {
	var out []byte
	n := jj.StreamParse(json, func(start, end, info int) int {
		out = append(out, json[start:end]...)
		return 1
	})
	if n != len(json) {
		panic(fmt.Sprintf("expected %d, got %d", len(json), n))
	}
	json = jj.Ugly(json)
	out = jj.Ugly(json)
	if string(out) != string(json) {
		panic("mismatch")
	}
}

// testFile tests if a JSON file is good
func testFile(path string) {
	json, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	mustBeAGood(json)
}

func TestFiles(t *testing.T) {
	fis, err := os.ReadDir("testdata/pjson")
	if err != nil {
		panic(err)
	}
	for _, fi := range fis {
		testFile(filepath.Join("testdata/pjson", fi.Name()))
	}
}

// lotsaOps preforms lots of operations and prints the results.
func lotsaOps(tag string, N int, op func() int) {
	start := time.Now()
	fmt.Printf("%-24s ", tag)
	var total int64
	for i := 0; i < N; i++ {
		total += int64(op())
	}
	var out bytes.Buffer
	jj.WriteOutput(&out, N, 1, time.Since(start), 0)
	fmt.Printf("%s, %.2f GB/sec\n", strings.TrimSpace(out.String()),
		float64(total)/time.Since(start).Seconds()/1024/1024/1024)
}

func testSpeed(path string) {
	baseName := filepath.Base(path)

	defer fmt.Printf("\n")
	var jdata []byte
	if baseName == "random-numbers.json" {
		jdata = makeRandomNumbersJSON()
	} else {
		var err error
		jdata, err = os.ReadFile(path)
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("== %s == (%d bytes)\n", baseName, len(jdata))
	N := 200000000 / len(jdata) / 10 * 10
	lotsaOps("jj.Parse (noop iter)", N, func() int {
		if jj.StreamParse(jdata, func(start, end, info int) int {
			return 1
		}) < 0 {
			panic("invalid")
		}
		return len(jdata)
	})
	lotsaOps("jj.Parse (nil iter)", N, func() int {
		if jj.StreamParse(jdata, nil) < 0 {
			panic("invalid")
		}
		return len(jdata)
	})
	lotsaOps("json.Valid (stdlib)", N, func() int {
		if !json.Valid(jdata) {
			panic("invalid")
		}
		return len(jdata)
	})
}

func TestSpeed(t *testing.T) {
	if os.Getenv("SPEED_TEST") == "" {
		fmt.Printf("Speed test disabled. Use SPEED_TEST=1\n")
		return
	}
	fmt.Printf("%s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fis, err := os.ReadDir("testfiles")
	if err != nil {
		panic(err)
	}
	for _, fi := range fis {
		t.Run(fi.Name(), func(t *testing.T) {
			testSpeed(filepath.Join("testfiles", fi.Name()))
		})
	}
	t.Run("random-numbers.json", func(t *testing.T) {
		testSpeed(filepath.Join("testfiles", "random-numbers.json"))
	})
}

func makeRandomNumbersJSON() []byte {
	rand.Seed(time.Now().UnixNano())
	N := 10000
	var json []byte
	json = append(json, '[')
	for i := 0; i < N; i++ {
		if i > 0 {
			json = append(json, ',')
		}
		x := rand.Float64()
		switch rand.Int() % 5 {
		case 0:
			x *= 1
		case 1:
			x *= 10
		case 2:
			x *= 100
		case 3:
			x *= 1000
		case 4:
			x *= 10000
		}
		switch rand.Int() % 2 {
		case 0:
			x *= -1
		case 1:
			x *= +1
		}
		switch rand.Int() % 6 {
		case 0:
			json = strconv.AppendFloat(json, x, 'f', -1, 64)
		case 1:
			json = strconv.AppendFloat(json, x, 'f', 0, 64)
		case 2:
			json = strconv.AppendFloat(json, x, 'f', 2, 64)
		case 3:
			json = strconv.AppendFloat(json, x, 'f', 4, 64)
		case 4:
			json = strconv.AppendFloat(json, x, 'f', 8, 64)
		case 5:
			json = strconv.AppendFloat(json, x, 'e', 8, 64)
		}
	}
	json = append(json, ']')
	return json
}
