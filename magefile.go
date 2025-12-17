//go:build mage

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

// Default target to run when none is specified
var Default = Help

// Help displays available mage targets
func Help() error {
	fmt.Println("📖 qumo-relay - Native Binary Deployment with Nomad")
	fmt.Printf("   Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()
	fmt.Println("Available targets:")
	fmt.Println()
	fmt.Println("  🏝️  Nomad Orchestration:")
	fmt.Println("    mage nomad:agent  - Start Nomad agent in dev mode")
	fmt.Println("    mage nomad:up     - Build and deploy to Nomad")
	fmt.Println("    mage nomad:stop   - Stop Nomad job")
	fmt.Println("    mage nomad:status - Show job status")
	fmt.Println("    mage nomad:logs   - Show job logs")
	fmt.Println("    mage nomad:clean  - Clean Nomad artifacts")
	fmt.Println()
	fmt.Println("  🌐 Web Demo:")
	fmt.Println("    mage relay        - Start relay server only")
	fmt.Println("    mage web          - Start web demo (Vite dev server only)")
	fmt.Println("    mage webBuild     - Build web demo for production")
	fmt.Println("    mage webClean     - Clean web build artifacts")
	fmt.Println()
	fmt.Println("  🔧 Utilities:")
	fmt.Println("    mage clean        - Clean build artifacts")
	fmt.Println()
	fmt.Println("  ℹ️  Info:")
	fmt.Println("    mage -l           - List all targets")
	fmt.Println("    mage help         - Show this help")
	fmt.Println()
	return nil
}

// Relay starts the qumo-relay server
func Relay() error {
	fmt.Println("📡 Starting qumo-relay server...")
	fmt.Println("   Relay: https://localhost:4433")
	fmt.Println("   Health: http://localhost:8080/health")
	fmt.Println()

	cmd := exec.Command("go", "run", "./cmd/qumo-relay")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Web starts the web demo application (Vite dev server only)
// Note: Start relay separately with `mage relay`
func Web() error {
	fmt.Println("🌐 Starting web demo...")
	fmt.Println("   Web Demo: http://localhost:5173")
	fmt.Println()
	fmt.Println("⚠️  Make sure relay is running separately:")
	fmt.Println("   mage relay")
	fmt.Println()

	// Start Vite dev server
	webDir := "web"
	cmd := exec.Command("deno", "task", "dev")
	cmd.Dir = webDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WebBuild builds the web demo for production
func WebBuild() error {
	fmt.Println("🔨 Building web demo...")

	cmd := exec.Command("deno", "task", "build")
	cmd.Dir = "web"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WebClean cleans web build artifacts
func WebClean() error {
	fmt.Println("🧹 Cleaning web artifacts...")
	return sh.Rm("web/dist")
}

// Clean removes build artifacts
func Clean() error {
	fmt.Println("🧹 Cleaning build artifacts...")
	return sh.Rm("bin")
}

// Nomad provides Nomad-specific commands
type Nomad mg.Namespace

// Agent starts the Nomad agent in dev mode
func (Nomad) Agent() error {
	fmt.Println("🏃 Starting Nomad Agent (Dev Mode)...")
	fmt.Println("   Access UI at http://localhost:4646")

	cmd := exec.Command("nomad", "agent", "-dev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Build builds the Go binary for Nomad
func (Nomad) Build() error {
	fmt.Println("🔨 Building qumo-relay binary...")

	binaryName := "qumo-relay"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	// Ensure build directory exists
	if err := os.MkdirAll("bin", 0755); err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-o", "bin/"+binaryName, "./cmd/qumo-relay")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Up builds and deploys the job to Nomad
func (Nomad) Up() error {
	mg.Deps(Nomad.Build)

	fmt.Println("🚀 Submitting Job to Nomad...")
	cmd := exec.Command("nomad", "job", "run", "moq-relay.nomad")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Stop stops the Nomad job
func (Nomad) Stop() error {
	fmt.Println("🛑 Stopping Nomad job...")
	cmd := exec.Command("nomad", "job", "stop", "moq-relay")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Status shows the Nomad job status
func (Nomad) Status() error {
	fmt.Println("📊 Job Status:")
	cmd := exec.Command("nomad", "job", "status", "moq-relay")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Logs shows the Nomad job logs
func (Nomad) Logs() error {
	fmt.Println("📋 Job Logs:")
	cmd := exec.Command("nomad", "alloc", "logs", "-job", "moq-relay", "-f")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Clean removes Nomad artifacts
func (Nomad) Clean() error {
	fmt.Println("🧹 Cleaning Nomad artifacts...")
	// Optionally stop the job first
	_ = Nomad{}.Stop()
	time.Sleep(1 * time.Second)
	return sh.Rm("bin")
}
