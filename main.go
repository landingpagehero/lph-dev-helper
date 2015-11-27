package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/wellington/go-libsass/libs"
	"gopkg.in/fsnotify.v1"
)

func main() {
	CompileAllFiles()
	CreateWatcher()
}

func CompileAllFiles() {
	files, err := doublestar.Glob("**/*.scss")

	if err != nil {
		return
	}

	for _, file := range files {
		if IsScss(file) {
			CompileScssAndWriteToCssFile(file)
		}
	}
}

func CompileScssAndWriteToCssFile(scssPath string) {
	compiledCss, err := CompileScss(scssPath)
	if err != nil {
		log.Fatal("error:", err)
		os.Exit(1)
	}

	cssPath := strings.Replace(scssPath, ".scss", ".css", 1)
	ioutil.WriteFile(cssPath, []byte(compiledCss), 0644)
	log.Println("compiled " + scssPath + " to " + cssPath)
}

func CompileScss(scssPath string) (compiledCss string, err error) {
	fileContent := libs.SassMakeFileContext(scssPath)
	options := libs.SassFileContextGetOptions(fileContent)

	libs.SassOptionSetOutputStyle(options, 2)
	libs.SassFileContextSetOptions(fileContent, options)

	context := libs.SassFileContextGetContext(fileContent)
	compiler := libs.SassMakeFileCompiler(fileContent)
	defer libs.SassDeleteCompiler(compiler)

	libs.SassCompilerParse(compiler)
	libs.SassCompilerExecute(compiler)

	compiledCss = libs.SassContextGetOutputString(context)

	errStatus := libs.SassContextGetErrorStatus(context)
	if errStatus > 0 {
		return "", errors.New(libs.SassContextGetErrorJSON(context))
	}

	return compiledCss, nil
}

func IsScss(path string) bool {
	return strings.HasSuffix(path, ".scss")
}

func CreateWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					if IsScss(event.Name) {
						log.Println("saw SCSS file created:", event.Name)
						CompileScssAndWriteToCssFile(event.Name)
					}
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					if IsScss(event.Name) {
						log.Println("saw SCSS file modified:", event.Name)
						CompileScssAndWriteToCssFile(event.Name)
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	if FileExists("./styles") {
		log.Println("watching styles/ directory for SCSS file changes")
		err = watcher.Add("./styles")
	} else {
		log.Println("no styles/ directory - not watching for SCSS file changes")
	}

	if err != nil {
		log.Fatal(err)
	}

	<-done
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}
