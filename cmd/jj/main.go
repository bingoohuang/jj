package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/bingoohuang/jj"
	"github.com/mattn/go-isatty"
)

var (
	version = "1.0.1"
	tag     = "jj - JSON Stream Editor " + version
	usage   = `usage: jj [-v value] [-curnOD] [-i infile] [-o outfile] keypath
eg.: jj keypath                      read value from stdin
     jj -i infile keypath            read value from infile
     jj -v value keypath             edit value
     jj -v value -o outfile keypath  edit value and write to outfile
options:
     -v value   Edit JSON key path value
     -c         Print cheatsheet
     -u         Make json ugly, keypath is optional
     -R         Create a random json
     -r         Use raw values, otherwise types are auto-detected
     -n         Do not modifyOutput color or extra formatting
     -O         Performance boost for value updates
     -D         Delete the value at the specified key path
     -l         Output array values on multiple lines
     -I         Print each child of json array
     -i infile  Use input file instead of stdin
     -g         Generate random JSON by input
     -e         Eval keypath value as an expression
     -p         Parse inner JSON string as a JSON
     -o outfile Use modifyOutput file instead of stdout
     -f regex   List the key and values which regex matches its key
     -k keypath JSON key path (like "name.last")
     -K keypath JSON key path as raw whole key
      keypath   Last argument for JSON key path`
)

type args struct {
	infile, outfile, value *string

	keypath, findRegex string

	raw, del, opt, keypathok, random      bool
	ugly, notty, lines, rawKey, gen, expr bool
	iterateArray, parseInnerJSONString    bool

	jsonMap map[string]interface{}
}

func parseArgs() args {
	fail := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, "%s\n", tag)
		if format != "" {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
		fmt.Fprintf(os.Stderr, "%s\n", usage)
		os.Exit(1)
	}
	help := func() {
		buf := &bytes.Buffer{}
		fmt.Fprintf(buf, "%s\n", tag)
		fmt.Fprintf(buf, "%s\n", usage)
		os.Stdout.Write(buf.Bytes())
		os.Exit(0)
	}

	var a args
	a.jsonMap = make(map[string]interface{})

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		default:
			if len(os.Args[i]) > 1 && os.Args[i][0] == '-' {
				for j := 1; j < len(os.Args[i]); j++ {
					switch os.Args[i][j] {
					case 'c':
						printCheatsAndExit()
					case 'u':
						a.ugly = true
					case 'r':
						a.raw = true
					case 'R':
						a.random = true
					case 'O':
						a.opt = true
					case 'D':
						a.del = true
					case 'n':
						a.notty = true
					case 'p':
						a.parseInnerJSONString = true
					case 'I':
						a.iterateArray = true
					case 'l':
						a.lines = true
					case 'g':
						a.gen = true
					case 'e':
						a.expr = true
					default:
						goto P1
					}
				}
				continue
			}
		P1:
			if p1 := strings.Index(os.Args[i], ":="); p1 > 0 {
				// Raw JSON fields
				a.jsonMap[os.Args[i][:p1]] = json.RawMessage(os.Args[i][p1+2:])
			} else if p2 := strings.Index(os.Args[i], "="); p2 > 0 {
				// Json fields
				a.jsonMap[os.Args[i][:p2]] = os.Args[i][p2+1:]
			} else if !a.keypathok {
				a.keypathok = true
				a.keypath = os.Args[i]
			} else {
				fail("unknown option argument: \"%s\"", a.keypath)
			}
		case "-v", "-i", "-o", "-k", "-K", "-f":
			arg := os.Args[i]
			i++
			if i >= len(os.Args) {
				fail("argument missing after: \"%s\"", arg)
			}
			switch arg {
			case "-v":
				a.value = &os.Args[i]
			case "-i":
				a.infile = &os.Args[i]
			case "-o":
				a.outfile = &os.Args[i]
			case "-k", "-K":
				a.keypathok = true
				a.keypath = os.Args[i]
				a.rawKey = arg == "-K"
			case "-f":
				a.findRegex = os.Args[i]
			}
		case "--force-notty":
			a.notty = true
		case "--version":
			fmt.Fprintf(os.Stdout, "%s\n", tag)
			os.Exit(0)
		case "-h", "--help", "-?":
			help()
		}
	}

	return a
}

var (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
)

func init() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
	}
}

//go:embed cheat.txt
var cheatText string

func init() {
	cheatText = strings.ReplaceAll(cheatText, "=>", Green+"=>"+Reset)
	cheatText = strings.ReplaceAll(cheatText, "$ ", Purple+"$ "+Reset)
	cheatText = strings.ReplaceAll(cheatText, " jj", Cyan+" jj"+Reset)
	for i := 30; i > 0; i-- {
		num := fmt.Sprintf("%d. ", i)
		cheatText = strings.Replace(cheatText, num, Red+num+Reset, 1)
	}
}

func printCheatsAndExit() {
	fmt.Println(cheatText)
	os.Exit(0)
}

func main() {
	a := parseArgs()
	f := a.createOutFile()

	outChan := make(chan Out)
	go a.createOut(outChan)

	for out := range outChan {
		outData := a.modifyOutput(f, out)
		_, _ = f.Write(outData)
	}
	_ = f.Close()
}

type Out struct {
	Data    []byte
	IsArray bool
	Type    jj.Type
}

func (a args) createOut(outChan chan Out) {
	if a.random {
		var out Out
		out.Data = jj.Rand()
		outChan <- out
		close(outChan)
		return
	}

	var input []byte
	var err error
	if len(a.jsonMap) > 0 {
		input, err = json.Marshal(a.jsonMap)
	} else {
		input, err = createInput(a)
	}
	if err != nil {
		fail(err)
	}

	if a.gen {
		a.generate(outChan, input)
		return
	}

	if a.iterateArray {
		a.doIterateArray(outChan, input)
		return
	}

	if a.parseInnerJSONString {
		a.formatInnerJsonString(outChan, input)
		return
	}

	if a.findRegex != "" {
		a.findKeyValues(input)
		close(outChan)
		return
	}

	opts := jj.SetOptions{PathOption: jj.PathOption{RawPath: a.rawKey}}

	if a.del {
		var out Out
		if out.Data, err = jj.DeleteBytes(input, a.keypath, opts); err != nil {
			fail(err)
		}
		outChan <- out
		close(outChan)
		return
	}

	if a.value != nil {
		raw := a.raw
		val := *a.value
		if !raw {
			switch val {
			default:
				if len(val) > 0 {
					if (val[0] >= '0' && val[0] <= '9') || val[0] == '-' {
						if _, err := strconv.ParseFloat(val, 64); err == nil {
							raw = true
						}
					}
				}
			case "true", "false", "null":
				raw = true
			}
		}

		if a.opt {
			opts.Optimistic = true
			opts.ReplaceInPlace = true
		}

		var out Out
		if raw { // set as raw block
			out.Data, err = jj.SetRawBytes(input, a.keypath, []byte(val), opts)
		} else { // set as a string
			out.Data, err = jj.SetBytes(input, a.keypath, val, opts)
		}
		if err != nil {
			fail(err)
		}

		outChan <- out
		close(outChan)
		return
	}

	var out Out
	if !a.keypathok {
		out.Data = input
	} else if a.expr {
		env := map[string]interface{}{}
		if err := json.Unmarshal(input, &env); err != nil {
			panic(err)
		}
		program, err := expr.Compile(a.keypath, expr.Env(env))
		if err != nil {
			panic(err)
		}

		output, err := expr.Run(program, env)
		if err != nil {
			panic(err)
		}
		v, err := json.Marshal(output)
		if err != nil {
			fail(err)
		}
		a.assignOut(&out, jj.ParseBytes(v))
	} else {
		res := jj.GetBytes(input, a.keypath, jj.WithRawPath(a.rawKey))
		a.assignOut(&out, res)
	}

	outChan <- out
	close(outChan)
	return
}

func (a args) formatInnerJsonString(outChan chan Out, input []byte) {
	out := jj.ParseBytes(jj.FreeInnerJSON(input))
	outChan <- Out{
		Data:    []byte(out.String()),
		IsArray: out.IsArray(),
		Type:    out.Type,
	}
	close(outChan)
}

func (a args) doIterateArray(outChan chan Out, input []byte) {
	started := false
	openCount := 0
	isArray := false
	elemStart := 0
	jj.StreamParse(input, func(start, end, info int) int {
		if !started {
			started = true
			isArray = jj.IsToken(info, jj.TokArray)
			return 1
		}

		if !isArray {
			return -1
		}

		if jj.IsToken(info, jj.TokOpen) {
			openCount++
			if openCount == 1 {
				elemStart = start
			}
		} else if jj.IsToken(info, jj.TokClose) {
			if openCount == 1 {
				out := Out{Data: input[elemStart:end], IsArray: jj.IsToken(info, jj.TokArray), Type: jj.JSON}
				outChan <- out
			}
			openCount--
		}

		if openCount == 0 && jj.IsToken(info, jj.TokString, jj.TokNumber, jj.TokTrue, jj.TokFalse, jj.TokNull) {
			var typ jj.Type
			switch {
			case jj.IsToken(info, jj.TokString):
				typ = jj.String
			case jj.IsToken(info, jj.TokNumber):
				typ = jj.Number
			case jj.IsToken(info, jj.TokTrue):
				typ = jj.True
			case jj.IsToken(info, jj.TokFalse):
				typ = jj.False
			case jj.IsToken(info, jj.TokNull):
				typ = jj.Null
			}

			out := Out{Data: input[start:end], IsArray: false, Type: typ}
			outChan <- out
		}

		return -1
	})

	close(outChan)
}

func (a args) findKeyValues(input []byte) {
	re := regexp.MustCompile(a.findRegex)
	foundKey := ""
	found := false
	openCount := 0
	openStart := 0
	jj.StreamParse(input, func(start, end, info int) int {
		v := string(input[start:end])
		if found {
			if jj.IsToken(info, jj.TokOpen) {
				openCount++
				if openCount == 1 {
					openStart = start
				}
			} else if jj.IsToken(info, jj.TokClose) {
				openCount--
				if openCount == 0 {
					fmt.Printf("%s: %s\n", foundKey, input[openStart:end])
					found = false
				}
			} else if openCount == 0 && jj.IsToken(info, jj.TokString, jj.TokNumber, jj.TokTrue, jj.TokFalse, jj.TokNull) {
				fmt.Printf("%s: %s\n", foundKey, v)
				found = false
			}
		} else if jj.IsToken(info, jj.TokKey) {
			k := v[1 : len(v)-1] // 去除两端双引号
			if found = re.MatchString(k); found {
				foundKey = v
			}
		}

		return 1
	})
}

func (a args) assignOut(out *Out, res jj.Result) {
	if a.raw {
		out.Data = []byte(res.Raw)
	} else {
		out.Type = res.Type
		out.IsArray = res.IsArray()
		out.Data = []byte(res.String())
	}
}

func (a args) generate(outChan chan Out, input []byte) {
	gen := jj.NewGen()
	s := string(input)
	for {
		genResult, i := gen.Process(s)
		if i <= 0 {
			break
		}

		var out Out
		out.Data = []byte(genResult.Out)
		outChan <- out
		s = s[i:]
	}

	close(outChan)
}

func createInput(a args) ([]byte, error) {
	if a.infile == nil {
		return ioutil.ReadAll(os.Stdin)
	} else {
		return ioutil.ReadFile(*a.infile)
	}
}

func (a args) createOutFile() *os.File {
	if a.outfile == nil {
		return os.Stdout
	}

	f, err := os.Create(*a.outfile)
	if err != nil {
		fail(err)
	}
	return f
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err.Error())
	os.Exit(1)
}

func (a args) modifyOutput(f *os.File, out Out) []byte {
	if a.lines && out.IsArray {
		var outb2 []byte
		jj.ParseBytes(out.Data).ForEach(func(_, v jj.Result) bool {
			outb2 = append(outb2, jj.Ugly([]byte(v.Raw))...)
			outb2 = append(outb2, '\n')
			return true
		})
		out.Data = outb2
	} else if a.raw || out.Type != jj.String {
		if a.ugly {
			out.Data = jj.Ugly(out.Data)
		} else {
			out.Data = jj.Pretty(out.Data)
		}
	}

	for len(out.Data) > 0 && out.Data[len(out.Data)-1] == '\n' {
		out.Data = out.Data[:len(out.Data)-1]
	}

	if !a.notty && isatty.IsTerminal(f.Fd()) {
		if a.raw || out.Type != jj.String {
			out.Data = jj.Color(out.Data, jj.TerminalStyle)
		} else {
			out.Data = append([]byte(jj.TerminalStyle.String[0]), out.Data...)
			out.Data = append(out.Data, jj.TerminalStyle.String[1]...)
		}
	}

	return append(out.Data, '\n')
}
