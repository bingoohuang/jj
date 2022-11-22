package jj

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/bingoohuang/jiami"
	"github.com/cristalhq/base64"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	aesKey          *jiami.Key
	encoder         *jiami.AesGcm
	encoderInitOnce sync.Once
)

func invokeJiami(result interface{}, wrapper string) interface{} {
	if wrapper != "..jiami" {
		return result
	}

	encoderInitOnce.Do(initJiami)

	plain := &jiami.Plain{}
	if str, ok := result.(string); ok {
		plain.Data = []byte(str)
	} else if bb, ok := result.([]byte); ok {
		plain.Data = bb
	} else {
		plain.Data = []byte(fmt.Sprintf("%v", result))
	}
	encoded, err := encoder.Encrypt(aesKey, plain)
	if err != nil {
		log.Fatalf("encrypt failed: %v", err)
	}

	b, err := msgpack.Marshal(encoded)
	if err != nil {
		log.Fatalf("msgpack.Marshal failed: %v", err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func wrapJiami(f func(args string) interface{}, wrapper string) func(args string) interface{} {
	if wrapper == "" {
		return f
	}
	if wrapper == "..jiami" {
		encoderInitOnce.Do(initJiami)

		return func(args string) interface{} {
			result := f(args)
			return invokeJiami(result, wrapper)
		}

	}

	return f
}

func initJiami() {
	aesKey = &jiami.Key{
		Passphrase: []byte(os.Getenv("PASSPHRASE")),
	}
	if len(aesKey.Passphrase) == 0 {
		aesKey.Passphrase = []byte("314159")
	}
	if err := aesKey.Init(); err != nil {
		log.Fatalf("create key failed: %v", err)
	}

	encoder = jiami.NewAesGcm()
}
