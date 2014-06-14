package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"flag"
)

// http://lawlessguy.wordpress.com/2013/07/23/filling-a-slice-using-command-line-flags-in-go-golang/
type Strings []string
func (i *Strings) String() string {
	return fmt.Sprintf("%s", *i)
}
func (i *Strings) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func check(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

func expandValue(text string) string {
	return os.ExpandEnv(text)
}

func visit(path string, f os.FileInfo, err error) error {
	if f.IsDir() {
		return nil
	}
	fmt.Fprintln(os.Stdout, "Processing " + path)
	blob, ioerr := ioutil.ReadFile(path)
	check(ioerr)
	ioutil.WriteFile(path, []byte(expandValue(string(blob))), f.Mode())
	return nil
}

var paths Strings
func main() {
	flag.Var(&paths, "path", "the file path")
	flag.Parse()

	if len(paths) > 0 {
		for i := 0; i < len(paths); i++ {
			path := paths[i]
			fileinfo, _ := os.Stat(path)
			if fileinfo.IsDir() {
				err := filepath.Walk(path, visit)
				check(err)
			} else {
				fmt.Fprintln(os.Stdout, "Processing " + path)
				visit(path, fileinfo, nil)
			}
		}
	}

	if len(flag.Args()) > 0 {
		env := os.Environ()
		exeerr := syscall.Exec(flag.Args()[0], flag.Args(), env)
		check(exeerr) // not reachable
	}
}
