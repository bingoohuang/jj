package jj

import (
	"errors"
	"fmt"
	"testing"
)

func ExampleNew() {
	// with multiple config
	_ = NewJtp(WithMaxArrayLen(6),
		WithMaxDepth(7),
		WithMaxKeyLen(20), WithMaxStringLen(50))

	// with single config
	_ = NewJtp(WithMaxStringLen(25))
}

func ExampleVerify_VerifyBytes() {
	json := []byte(`{
	"simple_string": "hello word",
    "targets": [
      {
        "req_per_second": 5,
        "duration_of_time": 1,
		"utf8Key": "Hello, 世界",
        "request": {
          "endpoint": "https://httpbin.org/get",
          "http_method": "GET",
          "payload": {
            "username": "ankur",
            "password": "ananad"
          },
		  "array_value": [
				"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstv"
			],
          "additional_header": [
            {
              "header_key": "uuid",
              "header_value": [
                "1",
                "2"
              ]
            }
          ]
        }
      },
      {
        "req_per_second": 10,
        "duration_of_time": 1,
        "request": {
          "endpoint": "https://httpbin.org/post",
          "http_method": "POST",
          "payload": {
            "username": "ankur",
            "password": "ananad"
          },
          "additional_header": [
            {
              "header_key": "uuid",
              "header_value": [
                "1",
                "2",
				"3",
				"4",
				"5",
				"Hello, 世界"
              ]
            }
          ]
        }
      }
    ]
}
	`)

	verifier1 := NewJtp(WithMaxArrayLen(6),
		WithMaxDepth(7),
		WithMaxKeyLen(20), WithMaxStringLen(50))
	err := verifier1.VerifyBytes(json)

	verifier2 := NewJtp(WithMaxStringLen(25))
	err = verifier2.VerifyBytes(json)
	fmt.Println(err)
	//  Output: jtp.maxStringLenReached.Max-[25]-Allowed.Found-[47]: jtp.MalformedJSON
}

func TestIsValidateString1(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		str      string
		isString bool
	}{
		{str: `i ♥ u`, isString: false},
		{str: `"Example \u2764\ufe0f"`, isString: true},
		{str: `"Example \u2764\ufe0f`, isString: true},
		// first char should also return
	}
	for _, tc := range scenarios {
		t.Run(tc.str, func(t *testing.T) {
			_, ok := isValidateString([]byte(tc.str), 0)
			if ok != tc.isString {
				t.Errorf("Expected %v Got %v", tc.isString, ok)
			}
		})
	}
}

func TestValidStringLengthUTF8(t *testing.T) {
	t.Parallel()
	maxAllowed := 10
	scenarios := []struct {
		str []byte
		err error
	}{
		{
			str: []byte("Hello, 世界"),
			err: nil,
		},
		{
			str: []byte(`i ♥ u`),
			err: nil,
		},
		{
			str: []byte(`"Hello, World!"`),
			err: fmt.Errorf("jtp.maxStringLenReached.Max-[10]-Allowed.Found-[13]: jtp.MalformedJSON"),
		},
	}

	for _, tc := range scenarios {
		t.Run(string(tc.str), func(t *testing.T) {
			e := validateStringLen(tc.str, 0, len(tc.str), maxAllowed, stringValueLength)
			if tc.err == nil && e != nil {
				t.Errorf("Expected an nil error Got - %v", e)
			}
			if tc.err != nil && e == nil {
				t.Errorf("Expected an not nil error Got - nil")
			}
			if tc.err != nil && e != nil && e.Error() != tc.err.Error() {
				t.Errorf("Expected error to be %s Got %s", tc.err.Error(), e.Error())
			}
		})
	}
}

func TestIsValidArrayCase1(t *testing.T) {
	t.Parallel()
	maxChild := 2
	scenarios := []struct {
		name string
		arr  []byte
		err  error
		ok   bool
	}{
		{
			name: "array len 3",
			arr:  []byte(`["Hello, 世界", "hello, world", "hi there"]`),
			err:  fmt.Errorf("jtp.maxArrayLenReached.Max-[2]-Allowed.Found-[3]: jtp.MalformedJSON"),
			ok:   false,
		},
		{
			name: "array len 2",
			arr:  []byte(`["Hello, 世界", "hi there"]`),
			err:  nil,
			ok:   true,
		},
		{
			name: "invalid array",
			arr:  []byte(`["Hello, 世界", "hi there"`),
			err:  nil,
			ok:   false,
		},
	}
	verifier := Verify{
		MaxArrayLen: maxChild,
	}
	var depth int
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			_, ok, err := verifier.isValidArray(tc.arr, 1, &depth)
			if tc.ok != ok {
				t.Errorf("Expected validation %v Got %v", tc.ok, ok)
			}
			if tc.err == nil && err != nil {
				t.Errorf("Expected an not nil error Got - nil")
			}
			if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
				t.Errorf("Expected error to be %s Got %s", tc.err.Error(), err.Error())
			}
		})
	}
}

func TestIsValidObjectCase1(t *testing.T) {
	t.Parallel()
	b := _getTestJSONBytes()
	scenarios := []struct {
		name     string
		verifier Verify
		err      error
		ok       bool
	}{
		{
			name:     "array max length 4",
			verifier: Verify{MaxArrayLen: 4},
			err:      fmt.Errorf("jtp.maxArrayLenReached.Max-[4]-Allowed.Found-[5]: jtp.MalformedJSON"),
			ok:       false,
		},
		{
			name:     "string key Length max 45",
			verifier: Verify{MaxStringLen: 45},
			err:      fmt.Errorf("jtp.maxStringLenReached.Max-[45]-Allowed.Found-[47]: jtp.MalformedJSON"),
			ok:       false,
		},
		{
			name:     "Object Key Length max 7",
			verifier: Verify{MaxKeyLen: 7},
			err:      fmt.Errorf("jtp.maxKeyLenReached.Max-[7]-Allowed.Found-[13]: jtp.MalformedJSON"),
			ok:       false,
		},
		{
			name:     "Object Key Length max 7",
			verifier: Verify{MaxKeyLen: 7},
			err:      fmt.Errorf("jtp.maxKeyLenReached.Max-[7]-Allowed.Found-[13]: jtp.MalformedJSON"),
			ok:       false,
		},
		{
			name:     "container depth 2",
			verifier: Verify{MaxDepth: 2},
			err:      fmt.Errorf("jtp.maxDepthReached.Max-[2]-Allowed.Found-[3]: jtp.MalformedJSON"),
			ok:       false,
		},
		{
			name: "container depth 5",
			verifier: Verify{
				MaxDepth: 5,
			},
			err: fmt.Errorf("jtp.maxDepthReached.Max-[5]-Allowed.Found-[6]: jtp.MalformedJSON"),
			ok:  false,
		},
		{
			name:     "Object Entry Count 4",
			verifier: Verify{MaxEntryCount: 4},
			err:      fmt.Errorf("jtp.maxEntryCountReached.Max-[4]-Allowed.Found-[5]: jtp.MalformedJSON"),
			ok:       false,
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			var depth int
			_, ok, err := tc.verifier.isValidObject(b, 1, &depth)
			if tc.ok != ok {
				t.Errorf("Expected validation %v Got %v", tc.ok, ok)
			}
			if tc.err == nil && err != nil {
				t.Errorf("Expected an not nil error Got - nil")
			}
			if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
				t.Errorf("Expected error to be %s Got %s", tc.err.Error(), err.Error())
			}
		})
	}
}

func TestTestifyNoJSONThreatInBytesErrorCase(t *testing.T) {
	t.Parallel()
	b := _getTestJSONBytes()
	scenarios := []struct {
		name     string
		verifier Verify
		err      error
	}{
		{
			name:     "array max length 4",
			verifier: Verify{MaxArrayLen: 4},
			err:      fmt.Errorf("jtp.maxArrayLenReached.Max-[4]-Allowed.Found-[5]: jtp.MalformedJSON"),
		},
		{
			name:     "string key Length max 45",
			verifier: Verify{MaxStringLen: 45},
			err:      fmt.Errorf("jtp.maxStringLenReached.Max-[45]-Allowed.Found-[47]: jtp.MalformedJSON"),
		},
		{
			name:     "Object Key Length max 7",
			verifier: Verify{MaxKeyLen: 7},
			err:      fmt.Errorf("jtp.maxKeyLenReached.Max-[7]-Allowed.Found-[13]: jtp.MalformedJSON"),
		},
		{
			name:     "Object Key Length max 7",
			verifier: Verify{MaxKeyLen: 7},
			err:      fmt.Errorf("jtp.maxKeyLenReached.Max-[7]-Allowed.Found-[13]: jtp.MalformedJSON"),
		},
		{
			name:     "container depth 2",
			verifier: Verify{MaxDepth: 2},
			err:      fmt.Errorf("jtp.maxDepthReached.Max-[2]-Allowed.Found-[3]: jtp.MalformedJSON"),
		},
		{
			name:     "container depth 5",
			verifier: Verify{MaxDepth: 5},
			err:      fmt.Errorf("jtp.maxDepthReached.Max-[5]-Allowed.Found-[6]: jtp.MalformedJSON"),
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.verifier.VerifyBytes(b)
			if tc.err == nil && err != nil {
				t.Errorf("Expected an not nil error Got - nil")
			}
			if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
				t.Errorf("Expected error to be %s Got %s", tc.err.Error(), err.Error())
			}
		})
	}
}

func TestTestifyNoJSONThreatInBytesErrorCase2(t *testing.T) {
	t.Parallel()
	b := _getMalformedTestJSONBytes()
	v := Verify{}

	t.Run("malformed json", func(t *testing.T) {
		err := v.VerifyBytes(b)
		if !errors.Is(err, ErrInvalidJSON) {
			t.Errorf("Expected Ok to Be False and Error of kind ErrInvalidJSON")
		}
	})
}

func TestTestifyNoJSONThreatInBytesPositiveCase1(t *testing.T) {
	t.Parallel()
	b := _getTestJSONBytes()
	v := Verify{}

	t.Run("Positive case 1", func(t *testing.T) {
		err := v.VerifyBytes(b)
		if err != nil {
			t.Errorf("Expected Ok to Be True and Error nil")
		}
	})
}

func TestTestifyNoJSONThreatInBytesPositiveBoundaryCase1(t *testing.T) {
	t.Parallel()
	b := _getTestJSONBytes()
	v := Verify{
		MaxArrayLen:   6,
		MaxDepth:      7,
		MaxKeyLen:     19,
		MaxStringLen:  50,
		MaxEntryCount: 5,
	}

	t.Run("PositiveBoundaryCase1", func(t *testing.T) {
		err := v.VerifyBytes(b)
		if err != nil {
			t.Errorf("Expected Ok to Be True and Error nil")
		}
	})
}

func TestTestifyNoJSONThreatInBytesPositiveBoundaryCase2(t *testing.T) {
	t.Parallel()
	b := _getTestJSONBytes()
	verifier := NewJtp(WithMaxArrayLen(6),
		WithMaxDepth(7),
		WithMaxKeyLen(19), WithMaxStringLen(50),
		WithMaxEntryCount(5))
	t.Run("with functional option parameter", func(t *testing.T) {
		err := verifier.VerifyBytes(b)
		if err != nil {
			t.Errorf("Expected Ok to Be True and Error nil")
		}
	})
}

func BenchmarkTestifyNoThreatInBytes(b *testing.B) {
	json := _getTestJSONBytes()
	verifier := NewJtp(WithMaxArrayLen(6),
		WithMaxDepth(7),
		WithMaxKeyLen(20), WithMaxStringLen(50),
		WithMaxEntryCount(5))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = verifier.VerifyBytes(json)
	}
}

func _getTestJSONBytes() []byte {
	return []byte(`{
	"simple_string": "hello word",
    "targets": [
      {
        "req_per_second_1": 5,
        "duration_of_time": 1,
		"utf8Key_1": "Hello, 世界",
        "request_1": {
          "endpoint": "https://httpbin.org/get",
          "http_method": "GET",
          "payload": {
            "username": "ankur",
            "password": "ananad"
          },
		  "array_value_1": [
				"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstv"
			],
          "additional_header_1": [
            {
              "header_key": "uuid",
              "header_value": [
                "1",
                "2"
              ]
            }
          ]
        }
      },
      {
        "req_per_second": 10,
        "duration_of_time": 1,
        "request": {
          "endpoint": "https://httpbin.org/post",
          "http_method": "POST",
          "payload": {
            "username": "ankur",
            "password": "ananad"
          },
          "additional_header": [
            {
              "header_key": "uuid",
              "header_value": [
                "1",
                "2",
				"3",
				"4",
				"5",
				"Hello, 世界"
              ]
            }
          ]
        }
      }
    ]
}
	`)
}

func _getMalformedTestJSONBytes() []byte {
	return []byte(`{
	"simple_string": "hello word",
    "targets": [
      {
        "req_per_second": 5,
        "duration_of_time": 1,
		"utf8Key": "Hello, 世界",
        "request": {
          "endpoint": "https://httpbin.org/get",
          "http_method": "GET",
          "payload": {
            "username": "ankur",
            "password": "ananad"
          },
		  "array_value": [
				"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstv"
			],
          "additional_header": [
            {
              "header_key": "uuid",
              "header_value": [
                "1",
                "2"
            }
          ]
        }
      },
      {
        "req_per_second": 10,
        "duration_of_time": 1,
        "request": {
          "endpoint": "https://httpbin.org/post",
          "http_method": "POST",
          "payload": {
            "username": "ankur",
            "password": "ananad"
          },
          "additional_header": [
            {
              "header_key": "uuid",
              "header_value": [
                "1",
                "2",
				"3",
				"4",
				"5",
				"Hello, 世界"
              ]
            }
          ]
        }
      }
    ]
}
	`)
}
