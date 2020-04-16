// Copyright 2020 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmdline_test

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel_testing"
)

func TestMain(m *testing.M) {
	bazel_testing.TestMain(m, bazel_testing.Args{
		Main: `
-- BUILD.bazel --
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_test(
    name = "racy_test",
    srcs = [
        "noracy_test.go",
        "racy_test.go",
    ],
    embed = [":racy"],
)

go_library(
    name = "racy",
    srcs = ["racy.go"],
    importpath = "example.com/racy",
)

-- racy.go --
package racy

import "fmt"

func Race() {
	var x int
	done := make(chan struct{})
	go func() {
		x = 1
		close(done)
	}()
	x = 2
	<-done
	fmt.Println(x)
}

-- racy_test.go --
// +build race

package racy

import "testing"

func TestRace(t *testing.T) {
	Race()
}

-- noracy_test.go --
// +build !race

package racy

const Does Not = Compile
`,
	})
}

// TestRace checks that the --@io_bazel_rules_go//go/config:race flag controls
// whether a target is built in race mode.
func TestRace(t *testing.T) {
	t.Logf("PATH=%s", os.Getenv("PATH"))

	// The test should not build unless it's in race mode.
	err := bazel_testing.RunBazel("test", "-s", "//:racy_test")
	if err == nil {
		t.Fatal("running //:racy_test without flag: unexpected success")
	}
	var xErr *exec.ExitError
	if !errors.As(err, &xErr) {
		t.Fatalf("unexpected error (of type %[1]T): %[1]v", err)
	}
	if code := xErr.ExitCode(); !xErr.Exited() || code != bazel_testing.BUILD_FAILURE {
		t.Errorf("running //:racy_test without flag: unexpected error (code %d): %v", code, err)
	}

	// The test should fail in race mode.
	err = bazel_testing.RunBazel("test", "-s", "--@io_bazel_rules_go//go/config:race", "//:racy_test")
	if err == nil {
		t.Fatal("running //:racy_test with flag: unexpected success")
	}
	if !errors.As(err, &xErr) {
		t.Fatalf("unexpected error (of type %[1]T): %[1]v", err)
	}
	if code := xErr.ExitCode(); !xErr.Exited() || code != bazel_testing.TESTS_FAILED {
		t.Errorf("running //:racy_test without flag: unexpected error (code %d): %v", code, err)
	}
}
