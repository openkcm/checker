//go:build integration

package integration_test

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const binary = "checker"

var validConfig, appVersion string

func init() {
	var err error
	var dat []byte

	// read config file
	dat, err = os.ReadFile("../config.yaml")
	if err != nil {
		panic(err)
	}
	validConfig = string(dat)

	// read VERSION file
	dat, err = os.ReadFile("../VERSION")
	if err != nil {
		panic(err)
	}
	appVersion = strings.TrimSpace(string(dat))
}

func buildCommandsAndRunTests(m *testing.M, cmds ...string) int {
	for _, name := range cmds {
		cmd := exec.Command("go", "build", "-buildvcs=false", "-race", "-cover", "-o", name, "../cmd/"+name)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Printf("output: %s", output)
			log.Fatalf("error: %v", err)
		}
		defer os.Remove(name)
	}

	code := m.Run()
	return code
}

func TestMain(m *testing.M) {
	// put this in a function so we can use defer to clean up
	code := buildCommandsAndRunTests(m, binary)

	// exit with the code from the tests
	os.Exit(code)
}
