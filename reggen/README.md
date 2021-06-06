# Reg-gen

This package generates strings based on regular expressions, forked from [here](github.com/lucasjones/reggen)

Try it [here](https://lucasjones.github.io/reggen)

## Usage

```go
package main

import (
	"fmt"
	"github.com/lucasjones/reggen"
)

func main() {
	// generate a single string
	str, err := reggen.Generate("^[a-z]{5,10}@[a-z]{5,10}\\.(com|net|org)$", 10)
	if err != nil {
		panic(err)
	}
	fmt.Println(str)

	// create a reusable generator
	g, err := reggen.NewGenerator("[01]{5}")
	if err != nil {
		panic(err)
	}

	for i := 0; i < 5; i++ {
		// 10 is the maximum number of times star, range or plus should repeat
		// i.e. [0-9]+ will generate at most 10 characters if this is set to 10
		fmt.Println(g.Generate(10))
	}
}
```

Sample output:

```log
bxnpubwc@kwrdbvjic.com
11000
01010
01100
01111
01001
```

## Relative

1. [generating random strings from regular expressions](https://github.com/zach-klippenstein/goregen)
