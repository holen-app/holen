package main

import (
	"fmt"
	"os"
	"path"
)

func main() {
	fmt.Println(path.Base(os.Args[0]))
}
