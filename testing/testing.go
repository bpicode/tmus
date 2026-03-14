package testing

import (
	"os"
	"path"
	"runtime"
)

// The filename returned by runtime.Caller(0) one currently executing this function, which is testing.go in this case.
// We change the working directory to the parent directory, i.e. is the root of the project.
// This allows test files to use relative paths to access testdata and other resources in the project.
func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}
