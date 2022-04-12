package jj

import "bytes"

// FreeInnerJSON frees the inner JSON string to a real JSON object.
// like {"body":"{\n  \"_hl\": \"mockbin\"}"} to {"body":{"_hl":"mockbin"}}
func FreeInnerJSON(input []byte) []byte {
	var b bytes.Buffer

	parseInnerJSON(input, &b)

	return b.Bytes()
}

func parseInnerJSON(input []byte, b *bytes.Buffer) {
	StreamParse(input, func(start, end, info int) int {
		s := input[start:end]
		if IsToken(info, TokString) && ValidBytes(s) {
			if r1 := ParseBytes(s); r1.Type == String {
				if r2 := Parse(r1.Str); r2.Type == JSON {
					b.WriteString(r1.Str)
					return 1
				}
			}
		}
		b.Write(s)
		return 1
	})
}
