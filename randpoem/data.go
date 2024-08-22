package randpoem

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
	"log"

	"github.com/bingoohuang/ngg/ss"
)

func RandPoetryTang() string { return SliceRandItem(PoetryTangsLines) }
func RandSongci() string     { return SliceRandItem(SongciLines) }
func RandShijing() string    { return SliceRandItem(ShijingLines) }

func SliceRandItem(data []string) string {
	return data[ss.Rand().Intn(len(data))]
}

var (
	//go:embed poetryTang.txt.gz
	PoetryTangTxtGz []byte

	//go:embed shijing.txt.gz
	ShijingTxtGz []byte

	//go:embed songci.txt.gz
	SongciTxtGz []byte

	PoetryTangsLines = UnGzipLines(PoetryTangTxtGz)
	ShijingLines     = UnGzipLines(ShijingTxtGz)
	SongciLines      = UnGzipLines(SongciTxtGz)
)

func UnGzipLines(input []byte) []string {
	content := MustUnGzip(input)
	scanner := bufio.NewScanner(bytes.NewReader(content))

	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return lines
}

func MustUnGzip(input []byte) []byte {
	r, err := UnGzip(input)
	if err != nil {
		log.Fatal(err)
	}
	return r
}

func UnGzip(input []byte) ([]byte, error) {
	g, err := gzip.NewReader(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	defer g.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, g)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
