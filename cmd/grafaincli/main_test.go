package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/iov-one/weave/tmtest"
)

// taken from testdata/config/config.toml - rpc.laddr
const tmURL = "http://localhost:44444"

// privKeyHex is a hex-encoded private key of an account with tokens on the test server
const privKeyHex = "d34c1970ae90acf3405f2d99dcaca16d0c7db379f4beafcfdf667b9d69ce350d27f5fb440509dfa79ec883a0510bc9a9614c3d44188881f0c5e402898b4bf3c9"

// addr is the hex address of the account that corresponds to privKeyHex
const addr = "E28AE9A6EB94FC88B73EB7CBD6B87BF93EB9BEF0"

// appName is the name of the application
const appName = "grafaind"

func TestMain(m *testing.M) {
	code := runTestMain(m)
	os.Exit(code)
}

// we need to do setup in a separate function, so cleanup is properly called
// os.Exit(code) above will never call defer
func runTestMain(m *testing.M) int {
	var t mockAsserter

	home, cleanup := tmtest.SetupConfig(t, "testdata")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	appCleanup := RunGrafain(ctx, t, home)
	tmCleanup := tmtest.RunTendermint(ctx, t, home)

	defer appCleanup()
	defer tmCleanup()

	return m.Run()
}

func RunGrafain(ctx context.Context, t tmtest.TestReporter, home string) (cleanup func()) {
	t.Helper()

	bnsdpath, err := exec.LookPath(appName)
	if err != nil {
		if os.Getenv("FORCE_TM_TEST") != "1" {
			t.Skip("Bnsd binary not found. Set FORCE_TM_TEST=1 to fail this test.")
		} else {
			t.Fatalf("Bnsd binary not found. Do not set FORCE_TM_TEST=1 to skip this test.")
		}
	}

	cmd := exec.CommandContext(ctx, bnsdpath, "-home", home, "start")
	// log tendermint output for verbose debugging....
	if os.Getenv("TM_DEBUG") != "" {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Bnsd process failed: %s", err)
	}

	// Give tendermint time to setup.
	time.Sleep(2 * time.Second)
	t.Logf("Running %s pid=%d", bnsdpath, cmd.Process.Pid)

	// Return a cleanup function, that will wait for bnsd to stop.
	// We also auto-kill when the context is Done
	cleanup = func() {
		t.Logf("bnsd cleanup called")
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	go func() {
		<-ctx.Done()
		cleanup()
	}()
	return cleanup
}

// mockAsserter lets us use the assert calls even though we have testing.M not testing.T
type mockAsserter struct{}

var _ tmtest.TestReporter = mockAsserter{}

func (mockAsserter) Helper() {}
func (mockAsserter) Fatal(args ...interface{}) {
	msg := fmt.Sprint(args...)
	panic(msg)
}
func (mockAsserter) Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	panic(msg)
}
func (mockAsserter) Log(args ...interface{}) {
	fmt.Println(args...)
}
func (mockAsserter) Logf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println("")
}
func (m mockAsserter) Skip(args ...interface{}) {
	m.Log(args...)
	os.Exit(0)
}
func (m mockAsserter) Skipf(format string, args ...interface{}) {
	m.Logf(format, args...)
	os.Exit(0)
}
