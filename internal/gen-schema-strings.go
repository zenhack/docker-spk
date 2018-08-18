package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
)

var (
	SandstormSrc = flag.String("sandstorm-src", "", "Path to sandstorm source directory")
)

func chkfatal(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()

	fileMap := map[string][]byte{}

	dirPath := *SandstormSrc + "/src/sandstorm"
	fis, err := ioutil.ReadDir(dirPath)
	chkfatal(err)

	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".capnp") {
			data, err := ioutil.ReadFile(dirPath + "/" + fi.Name())
			chkfatal(err)
			fileMap[fi.Name()] = data
		}
	}
	fmt.Print("package main\n" +
		"\n" +
		"var CapnpFileMap = map[string]string{\n")
	for k, v := range fileMap {
		fmt.Printf("%q: %q,\n", k, v)
	}
	fmt.Print("}\n")
}
