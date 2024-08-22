package jj

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/bingoohuang/easyjson/jlexer"
	"github.com/buger/jsonparser"
	jsoni "github.com/json-iterator/go"
	fflib "github.com/pquerna/ffjson/fflib/v1"
)

type BenchStruct struct {
	Widget struct {
		Window struct {
			Name string `json:"name"`
		} `json:"window"`
		Image struct {
			HOffset int `json:"hOffset"`
		} `json:"image"`
		Text struct {
			OnMouseUp string `json:"onMouseUp"`
		} `json:"text"`
	} `json:"widget"`
}

var benchPaths = []string{
	"widget.window.name",
	"widget.image.hOffset",
	"widget.text.onMouseUp",
}

var benchManyPaths = []string{
	"widget.window.name",
	"widget.image.hOffset",
	"widget.text.onMouseUp",
	"widget.window.title",
	"widget.image.alignment",
	"widget.text.style",
	"widget.window.height",
	"widget.image.src",
	"widget.text.data",
	"widget.text.size",
}

func BenchmarkGJSONGet(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			if Get(exampleJSON, benchPaths[j]).Type == Null {
				t.Fatal("did not find the value")
			}
		}
	}
}

func BenchmarkGJSONGetMany4Paths(t *testing.B) {
	benchmarkGJSONGetManyN(t, 4)
}

func BenchmarkGJSONGetMany8Paths(t *testing.B) {
	benchmarkGJSONGetManyN(t, 8)
}

func BenchmarkGJSONGetMany16Paths(t *testing.B) {
	benchmarkGJSONGetManyN(t, 16)
}

func BenchmarkGJSONGetMany32Paths(t *testing.B) {
	benchmarkGJSONGetManyN(t, 32)
}

func BenchmarkGJSONGetMany64Paths(t *testing.B) {
	benchmarkGJSONGetManyN(t, 64)
}

func BenchmarkGJSONGetMany128Paths(t *testing.B) {
	benchmarkGJSONGetManyN(t, 128)
}

func benchmarkGJSONGetManyN(t *testing.B, n int) {
	var paths []string
	for len(paths) < n {
		paths = append(paths, benchManyPaths...)
	}
	paths = paths[:n]
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		results := GetMany(exampleJSON, paths...)
		if len(results) == 0 {
			t.Fatal("did not find the value")
		}
		for j := 0; j < len(results); j++ {
			if results[j].Type == Null {
				t.Fatal("did not find the value")
			}
		}
	}
}

func BenchmarkGJSONUnmarshalMap(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			parts := strings.Split(benchPaths[j], ".")
			m, _ := Parse(exampleJSON).Value().(map[string]any)
			var v any
			for len(parts) > 0 {
				part := parts[0]
				if len(parts) > 1 {
					m = m[part].(map[string]any)
					if m == nil {
						t.Fatal("did not find the value")
					}
				} else {
					v = m[part]
					if v == nil {
						t.Fatal("did not find the value")
					}
				}
				parts = parts[1:]
			}
		}
	}
}

func BenchmarkJSONUnmarshalMap(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			parts := strings.Split(benchPaths[j], ".")
			var m map[string]any
			if err := json.Unmarshal([]byte(exampleJSON), &m); err != nil {
				t.Fatal(err)
			}
			var v any
			for len(parts) > 0 {
				part := parts[0]
				if len(parts) > 1 {
					m = m[part].(map[string]any)
					if m == nil {
						t.Fatal("did not find the value")
					}
				} else {
					v = m[part]
					if v == nil {
						t.Fatal("did not find the value")
					}
				}
				parts = parts[1:]
			}
		}
	}
}

func BenchmarkJSONUnmarshalStruct(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			var s BenchStruct
			if err := json.Unmarshal([]byte(exampleJSON), &s); err != nil {
				t.Fatal(err)
			}
			switch benchPaths[j] {
			case "widget.window.name":
				if s.Widget.Window.Name == "" {
					t.Fatal("did not find the value")
				}
			case "widget.image.hOffset":
				if s.Widget.Image.HOffset == 0 {
					t.Fatal("did not find the value")
				}
			case "widget.text.onMouseUp":
				if s.Widget.Text.OnMouseUp == "" {
					t.Fatal("did not find the value")
				}
			}
		}
	}
}

func BenchmarkJSONDecoder(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			dec := json.NewDecoder(bytes.NewBuffer([]byte(exampleJSON)))
			var found bool
		outer:
			for {
				tok, err := dec.Token()
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Fatal(err)
				}
				switch v := tok.(type) {
				case string:
					if found {
						// break out once we find the value.
						break outer
					}
					switch benchPaths[j] {
					case "widget.window.name":
						if v == "name" {
							found = true
						}
					case "widget.image.hOffset":
						if v == "hOffset" {
							found = true
						}
					case "widget.text.onMouseUp":
						if v == "onMouseUp" {
							found = true
						}
					}
				}
			}
			if !found {
				t.Fatal("field not found")
			}
		}
	}
}

func BenchmarkFFJSONLexer(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			l := fflib.NewFFLexer([]byte(exampleJSON))
			var found bool
		outer:
			for {
				t := l.Scan()
				if t == fflib.FFTok_eof {
					break
				}
				if t == fflib.FFTok_string {
					b, _ := l.CaptureField(t)
					v := string(b)
					if found {
						// break out once we find the value.
						break outer
					}
					switch benchPaths[j] {
					case "widget.window.name":
						if v == "\"name\"" {
							found = true
						}
					case "widget.image.hOffset":
						if v == "\"hOffset\"" {
							found = true
						}
					case "widget.text.onMouseUp":
						if v == "\"onMouseUp\"" {
							found = true
						}
					}
				}
			}
			if !found {
				t.Fatal("field not found")
			}
		}
	}
}

func skipCC(l *jlexer.Lexer, n int) {
	for i := 0; i < n; i++ {
		l.Skip()
		l.WantColon()
		l.Skip()
		l.WantComma()
	}
}

func skipGroup(l *jlexer.Lexer, n int) {
	l.WantColon()
	l.Delim('{')
	skipCC(l, n)
	l.Delim('}')
	l.WantComma()
}

func easyJSONWindowName(t *testing.B, l *jlexer.Lexer) {
	if l.String() == "window" {
		l.WantColon()
		l.Delim('{')
		skipCC(l, 1)
		if l.String() == "name" {
			l.WantColon()
			if l.String() == "" {
				t.Fatal("did not find the value")
			}
		}
	}
}

func easyJSONImageHOffset(t *testing.B, l *jlexer.Lexer) {
	if l.String() == "image" {
		l.WantColon()
		l.Delim('{')
		skipCC(l, 1)
		if l.String() == "hOffset" {
			l.WantColon()
			if l.Int() == 0 {
				t.Fatal("did not find the value")
			}
		}
	}
}

func easyJSONTextOnMouseUp(t *testing.B, l *jlexer.Lexer) {
	if l.String() == "text" {
		l.WantColon()
		l.Delim('{')
		skipCC(l, 5)
		if l.String() == "onMouseUp" {
			l.WantColon()
			if l.String() == "" {
				t.Fatal("did not find the value")
			}
		}
	}
}

func easyJSONWidget(t *testing.B, l *jlexer.Lexer, j int) {
	l.WantColon()
	l.Delim('{')
	switch benchPaths[j] {
	case "widget.window.name":
		skipCC(l, 1)
		easyJSONWindowName(t, l)
	case "widget.image.hOffset":
		skipCC(l, 1)
		if l.String() == "window" {
			skipGroup(l, 4)
		}
		easyJSONImageHOffset(t, l)
	case "widget.text.onMouseUp":
		skipCC(l, 1)
		if l.String() == "window" {
			skipGroup(l, 4)
		}
		if l.String() == "image" {
			skipGroup(l, 4)
		}
		easyJSONTextOnMouseUp(t, l)
	}
}

func BenchmarkEasyJSONLexer(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			l := &jlexer.Lexer{Data: []byte(exampleJSON)}
			l.Delim('{')
			if l.String() == "widget" {
				easyJSONWidget(t, l, j)
			}
		}
	}
}

func BenchmarkJSONParserGet(t *testing.B) {
	data := []byte(exampleJSON)
	keys := make([][]string, 0, len(benchPaths))
	for i := 0; i < len(benchPaths); i++ {
		keys = append(keys, strings.Split(benchPaths[i], "."))
	}
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j, k := range keys {
			if j == 1 {
				// "widget.image.hOffset" is a number
				v, _ := jsonparser.GetInt(data, k...)
				if v == 0 {
					t.Fatal("did not find the value")
				}
			} else {
				// "widget.window.name",
				// "widget.text.onMouseUp",
				v, _ := jsonparser.GetString(data, k...)
				if v == "" {
					t.Fatal("did not find the value")
				}
			}
		}
	}
}

func jsoniterWindowName(t *testing.B, iter *jsoni.Iterator) {
	var v string
	for {
		key := iter.ReadObject()
		if key != "window" {
			iter.Skip()
			continue
		}
		for {
			key := iter.ReadObject()
			if key != "name" {
				iter.Skip()
				continue
			}
			v = iter.ReadString()
			break
		}
		break
	}
	if v == "" {
		t.Fatal("did not find the value")
	}
}

func jsoniterTextOnMouseUp(t *testing.B, iter *jsoni.Iterator) {
	var v string
	for {
		key := iter.ReadObject()
		if key != "text" {
			iter.Skip()
			continue
		}
		for {
			key := iter.ReadObject()
			if key != "onMouseUp" {
				iter.Skip()
				continue
			}
			v = iter.ReadString()
			break
		}
		break
	}
	if v == "" {
		t.Fatal("did not find the value")
	}
}

func jsoniterImageOffset(t *testing.B, iter *jsoni.Iterator) {
	var v int
	for {
		key := iter.ReadObject()
		if key != "image" {
			iter.Skip()
			continue
		}
		for {
			key := iter.ReadObject()
			if key != "hOffset" {
				iter.Skip()
				continue
			}
			v = iter.ReadInt()
			break
		}
		break
	}
	if v == 0 {
		t.Fatal("did not find the value")
	}
}

func jsoniterWidget(t *testing.B, iter *jsoni.Iterator, j int) {
	for {
		key := iter.ReadObject()
		if key != "widget" {
			iter.Skip()
			continue
		}
		switch benchPaths[j] {
		case "widget.window.name":
			jsoniterWindowName(t, iter)
		case "widget.image.hOffset":
			jsoniterImageOffset(t, iter)
		case "widget.text.onMouseUp":
			jsoniterTextOnMouseUp(t, iter)
		}
		break
	}
}

func BenchmarkJSONIterator(t *testing.B) {
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		for j := 0; j < len(benchPaths); j++ {
			iter := jsoni.ParseString(jsoni.ConfigDefault, exampleJSON)
			jsoniterWidget(t, iter, j)
		}
	}
}

var massiveJSON = func() string {
	var buf bytes.Buffer
	buf.WriteString("[")
	for i := 0; i < 100; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(exampleJSON)
	}
	buf.WriteString("]")
	return buf.String()
}()

func BenchmarkConvertNone(t *testing.B) {
	mj := massiveJSON
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		Get(mj, "50.widget.text.onMouseUp")
	}
}

func BenchmarkConvertGet(t *testing.B) {
	data := []byte(massiveJSON)
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		Get(string(data), "50.widget.text.onMouseUp")
	}
}

func BenchmarkConvertGetBytes(t *testing.B) {
	data := []byte(massiveJSON)
	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		GetBytes(data, "50.widget.text.onMouseUp")
	}
}
