package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type cliConfig struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

type humaConfig struct {
	Service      string     `yaml:"service"`
	Command      string     `yaml:"command"`
	CLI          *cliConfig `yaml:"cli"`
	SDKLanguages []string   `yaml:"sdk-languages"`
}

// Run a command capturing output and panics if there was an error.
func run(name string, args ...string) string {
	return runEnv(nil, name, args...)
}

// Run a command capturing output with a specific env setup. Panics on error.
func runEnv(env []string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("Error running command `%s %v`: %w\n%s", name, args, err, output))
	}
	return string(output)
}

func main() {
	if _, err := os.Stat(".huma.yaml"); err != nil {
		panic("cannot find required `.huma.yaml` file")
	}

	// Read YAML config.
	var config humaConfig
	data, err := ioutil.ReadFile(".huma.yaml")
	if err != nil {
		panic(err)
	}
	log.Println("Read .huma.yaml config file")

	if err := yaml.Unmarshal(data, &config); err != nil {
		panic(err)
	}

	// Set defaults.
	if config.Service == "" {
		panic("service name is required")
	}

	if config.Command == "" {
		config.Command = "go test && go install"
	}

	// Test & Install the service.
	log.Println("Building service " + config.Service)
	run("sh", "-c", config.Command)

	// Generate the OpenAPI 3 document.
	log.Println("Generating OpenAPI 3 spec")
	specPath := config.Service + ".json"
	run(config.Service, "openapi", specPath)

	// Get the version for CLI/SDKs.
	os.MkdirAll("out", 0755)
	out := run(config.Service, "--version")
	parts := strings.Split(out, " ")
	version := strings.TrimSpace(parts[len(parts)-1])
	log.Println("Service " + config.Service + " version is: " + version)

	// Build the CLI.
	if _, err := os.Stat("cli"); err == nil {
		if config.CLI == nil {
			config.CLI = &cliConfig{}
		}

		if config.CLI.Name == "" {
			config.CLI.Name = config.Service + "-cli"
		}

		if config.CLI.Command == "" {
			config.CLI.Command = "go build"
		}

		os.Chdir("cli")
		run("sh", "-c", "go generate")

		log.Println("Building CLI " + config.CLI.Name + "-windows-" + version)
		runEnv([]string{"GOOS=windows", "GOARCH=amd64"}, "sh", "-c", config.CLI.Command)
		z := config.CLI.Name + "-windows-" + version + ".zip"
		run("zip", "-r", z, config.CLI.Name+".exe")
		run("sh", "-c", "mv "+z+" ../out/")

		log.Println("Building CLI " + config.CLI.Name + "-mac-" + version)
		runEnv([]string{"GOOS=darwin", "GOARCH=amd64"}, "sh", "-c", config.CLI.Command)
		z = config.CLI.Name + "-mac-" + version + ".zip"
		run("zip", "-r", z, config.CLI.Name)
		run("sh", "-c", "mv "+z+" ../out/")

		log.Println("Building CLI " + config.CLI.Name + "-linux-" + version)
		runEnv([]string{"GOOS=linux", "GOARCH=amd64"}, "sh", "-c", config.CLI.Command)
		z = config.CLI.Name + "-linux-" + version + ".zip"
		run("zip", "-r", z, config.CLI.Name)
		run("sh", "-c", "mv "+z+" ../out/")

		os.Chdir("..")
	} else {
		log.Println("Skipping CLI build, no `cli` folder found.")
	}

	// Build the SDKs.
	for _, sdk := range config.SDKLanguages {
		sdkName := config.Service + "-" + sdk + "-" + version
		sdkPath := "out/" + sdkName

		log.Println("Building " + sdkName)

		// Generate the SDK.
		run("/usr/local/bin/docker-entrypoint.sh", "generate", "--enable-post-process-file", "-i", specPath, "-g", sdk, "-o", sdkPath)

		// Zip it up.
		run("sh", "-c", "cd out && zip -r "+sdkName+".zip "+sdkName)
	}
}
