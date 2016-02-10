package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/uiureo/jack/tokenizer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "no files given")
		os.Exit(1)
	}

	filename := os.Args[1]
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	tokens := tokenizer.Tokenize(string(data))

	for _, token := range tokens {
		fmt.Printf("%s %s\n", token.TokenType, token.Value)
	}
}
