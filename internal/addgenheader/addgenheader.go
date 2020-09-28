// addgenheader is a simple program that adds a DO NOT EDIT style
// comment at the top of a file. Because some generators do not do
// this, e.g. go-bindata
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// %v DO NOT EDIT\n", strings.TrimSpace(os.Args[2]))
	fmt.Fprintf(&buf, "\n")
	byts, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(&buf, "%s", byts)
	if err := ioutil.WriteFile(os.Args[1], buf.Bytes(), 0666); err != nil {
		panic(err)
	}
}
