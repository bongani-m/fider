// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// warning shown when at least one dependency is not installed
var missingDepsWarning = `Dependencies %v are missing. Please install them and try again.
To learn how, visit our contributors guide: https://github.com/getfider/fider/blob/master/CONTRIBUTING.md.
`

// required dependencies for building fider
var requiredDeps = []string{
	"air",
	"godotenv",
	"docker",
	"npm",
	"node",
	"mage",
	"golangci-lint",
}
var buildTime = time.Now().Format("2006.01.02.150405")
var buildNumber = os.Getenv("CIRCLE_BUILD_NUM")
var exeName = "fider"

var Aliases = map[string]interface{}{
	"build": Build.All,
	"test":  Test.All,
	"watch": Watch.All,
	"lint":  Lint.All,
}

func init() {
	os.Setenv("MAGEFILE_VERBOSE", "true")
	if runtime.GOOS == "windows" {
		exeName = "fider.exe"
	}

	missingDeps := missingDependencies()
	if len(missingDeps) > 0 {
		fmt.Printf(missingDepsWarning, missingDeps)
		os.Exit(1)
	}
}

func Run() error {
	return sh.Run("godotenv", "-f", ".env", "./"+exeName)
}

func Migrate() error {
	return sh.Run("godotenv", "-f", ".env", "./"+exeName, "migrate")
}

func Clean() error {
	os.RemoveAll("./dist")
	return os.Mkdir("./dist", 0777)
}

type Watch mg.Namespace

func (Watch) All() {
	mg.SerialDeps(Clean, Build.Server, Migrate)
	mg.Deps(Watch.Server, Watch.UI)
}

func (Watch) UI() error {
	return sh.Run("npx", "webpack-cli", "-w")
}

func (Watch) Server() error {
	return sh.Run("air", "-c", "air.conf")
}

type Build mg.Namespace

func (Build) All() {
	mg.Deps(Build.Server, Build.UI)
}

func (Build) Server() error {
	env := map[string]string{
		"GOOS":   runtime.GOOS,
		"GOARCH": runtime.GOARCH,
	}
	ldflags := "-s -w -X main.buildtime=" + buildTime + " -X main.buildnumber=" + buildNumber
	return sh.RunWith(env, "go", "build", "-ldflags", ldflags, "-o", exeName, ".")
}

func (Build) UI() error {
	mg.Deps(Clean)
	env := map[string]string{"NODE_ENV": "production"}
	return sh.RunWith(env, "npx", "webpack-cli")
}

type Test mg.Namespace

func (Test) All() {
	mg.Deps(Test.Server, Test.UI)
}

func (Test) Coverage() error {
	mg.Deps(Build.Server)
	sh.Run("godotenv", "-f", ".test.env", "./"+exeName, "migrate")
	return sh.Run("godotenv", "-f", ".test.env", "go", "test", "./...", "-coverprofile=cover.out", "-coverpkg=all", "-p=8", "-race")
}

func (Test) Server() error {
	mg.Deps(Build.Server)
	sh.Run("godotenv", "-f", ".test.env", "./"+exeName, "migrate")
	return sh.Run("godotenv", "-f", ".test.env", "go", "test", "./...", "-race")
}

func (Test) UI() error {
	env := map[string]string{"TZ": "GMT"}
	return sh.RunWith(env, "npx", "jest", "./public")
}

type Lint mg.Namespace

func (Lint) All() {
	mg.Deps(Lint.Server, Lint.UI)
}

func (Lint) UI() error {
	return sh.Run("npx", "tslint", "-c", "tslint.json", "'public/**/*.{ts,tsx}'", "'tests/**/*.{ts,tsx}'")
}

func (Lint) Server() error {
	return sh.Run("golangci-lint", "run")
}

// Utils
func missingDependencies() []string {
	var missingDeps []string
	for _, dep := range requiredDeps {
		_, err := exec.LookPath(dep)
		if err != nil {
			missingDeps = append(missingDeps, dep)
		}
	}
	return missingDeps
}
