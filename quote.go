package jj

import (
	"bytes"
	"fmt"
	"regexp"
	"unicode/utf8"
)

// FormatQuoteNameLeniently 将 JSON 中 key 的非必要双引号去除
func FormatQuoteNameLeniently(input []byte, optFns ...QuoteOptionFunc) []byte {
	var b bytes.Buffer
	formatQuoteNameLeniently(input, &b, optFns...)
	return b.Bytes()
}

type QuoteOption struct {
	LenientValue bool
}

type QuoteOptionFunc func(o *QuoteOption)

func WithLenientValue() QuoteOptionFunc {
	return func(o *QuoteOption) {
		o.LenientValue = true
	}
}

func formatQuoteNameLeniently(input []byte, b *bytes.Buffer, optFns ...QuoteOptionFunc) {
	var opt QuoteOption
	for _, fn := range optFns {
		fn(&opt)
	}

	StreamParse(input, func(start, end, info int) int {
		s := input[start:end]
		if IsToken(info, TokKey) || opt.LenientValue && IsToken(info, TokString) {
			k := s[1 : len(s)-1] // 去除两端双引号
			b.Write([]byte(QuoteNameLeniently(string(k))))
		} else {
			b.Write(s)
		}
		return 1
	})
}

// QuoteNameLeniently 对 name 如果需要的话，添加引号
// 取自: https://github.com/hjson/hjson-go/blob/master/encode.go
func QuoteNameLeniently(name string) string {
	if len(name) == 0 {
		return `""`
	}

	// Check if we can insert this name without quotes
	if needsEscapeName.MatchString(name) || needsEscape.MatchString(name) {
		return `"` + quoteReplace(name) + `"`
	}

	// without quotes
	return name
}

const commonRange = `\x7f-\x9f\x{00ad}\x{0600}-\x{0604}\x{070f}\x{17b4}\x{17b5}\x{200c}-\x{200f}\x{2028}-\x{202f}\x{2060}-\x{206f}\x{feff}\x{fff0}-\x{ffff}`

var (
	needsEscapeName = regexp.MustCompile(`[,{\[}\]\s:#"']|//|/\*`)

	// needsEscape tests if the string can be written without escapes
	needsEscape = regexp.MustCompile(`[\\\"\x00-\x1f` + commonRange + `]`)
)

var meta = map[byte][]byte{
	// table of character substitutions
	'\b': []byte(`\b`),
	'\t': []byte(`\t`),
	'\n': []byte(`\n`),
	'\f': []byte(`\f`),
	'\r': []byte(`\r`),
	'"':  []byte(`\"`),
	'\\': []byte(`\\`),
}

func quoteReplace(text string) string {
	return string(needsEscape.ReplaceAllFunc([]byte(text), func(a []byte) []byte {
		c := meta[a[0]]
		if c != nil {
			return c
		}
		r, _ := utf8.DecodeRune(a)
		return []byte(fmt.Sprintf("\\u%04x", r))
	}))
}
