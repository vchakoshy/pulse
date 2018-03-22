package pulser

import (
	"log"
	"path"
	"runtime"
)

func getRootDir() string {
	_, filename, _, _ := runtime.Caller(0)
	rootDir := path.Dir(path.Dir(filename))
	log.Println("root dir:", rootDir)
	return rootDir
}
