package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/wellington/go-libsass/libs"
	"gopkg.in/fsnotify.v1"
)

/**
 * If run without an argument, builds files in the current directory and watches for changes.
 * If run with the argument "build", builds files in the current directory and exits.
 * If run with the argument "watch", watches for changes in the current directory.
 */
func main() {
	if len(os.Args) > 1 {
		command := os.Args[1]
		if command == "build" {
			CompileAllFiles()
		} else if command == "watch" {
			CreateWatcher()
		} else {
			log.Fatal("unknown argument: " + command)
		}
	} else {
		CompileAllFiles()
		CreateWatcher()
	}
}

func CompileAllFiles() {
	files, err := doublestar.Glob("{styles,scripts}/*.{scss,js6}")

	if err != nil {
		return
	}

	for _, file := range files {
		if IsScss(file) {
			CompileScssAndWriteToCssFile(file)
		}
		if IsES6(file) {
			CompileES6AndWriteToJsFile(file)
		}
	}
}

func CompileScssAndWriteToCssFile(scssPath string) {
	compiledCss, err := CompileScss(scssPath)
	if err != nil {
		log.Fatal("error:", err)
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

func CompileES6AndWriteToJsFile(es6Path string) {
	jsPath := strings.Replace(es6Path, ".js6", ".js", 1)
	_, err := exec.Command("sh", "-c", "traceur --script "+es6Path+"  --out "+jsPath).Output()
	if err != nil {
		log.Fatal("error: ensure traceur is installed global (npm install -g traceur): error message was: ", err)
	}

	log.Println("compiled " + es6Path + " to " + jsPath)
}

func IsScss(path string) bool {
	return strings.HasSuffix(path, ".scss")
}

func IsES6(path string) bool {
	return strings.HasSuffix(path, ".js6")
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
					if IsES6(event.Name) {
						log.Println("saw JS/ES6 file created:", event.Name)
						CompileES6AndWriteToJsFile(event.Name)
					}
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					if IsScss(event.Name) {
						log.Println("saw SCSS file modified:", event.Name)
						CompileScssAndWriteToCssFile(event.Name)
					}
					if IsES6(event.Name) {
						log.Println("saw JS/ES6 file modified:", event.Name)
						CompileES6AndWriteToJsFile(event.Name)
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

	if FileExists("./scripts") {
		log.Println("watching scripts/ directory for JS file changes")
		err = watcher.Add("./scripts")
	} else {
		log.Println("no scripts/ directory - not watching for JS file changes")
	}

	if err != nil {
		log.Fatal(err)
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
