//go:build mage

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
var Default = Help

// Help displays available mage targets
func Help() error {
	fmt.Println("📖 qumo-relay - Docker E2E Test Automation")
	fmt.Printf("   Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()
	fmt.Println("Available targets:")
	fmt.Println()
	fmt.Println("  🧪 Testing:")
	fmt.Println("    mage e2e          - Run end-to-end tests")
	fmt.Println("    mage ci           - Run full CI pipeline (build → up → test → down)")
	fmt.Println()
	fmt.Println("  🐳 Docker:")
	fmt.Println("    mage build        - Build Docker image")
	fmt.Println("    mage up           - Start containers")
	fmt.Println("    mage down         - Stop containers")
	fmt.Println("    mage logs         - Show container logs")
	fmt.Println()
	fmt.Println("  🌐 Web Demo:")
	fmt.Println("    mage web          - Start web demo (relay + frontend)")
	fmt.Println("    mage webBuild     - Build web demo for production")
	fmt.Println("    mage webClean     - Clean web build artifacts")
	fmt.Println()
	if runtime.GOOS == "windows" {
		fmt.Println("  🐧 WSL (Windows only):")
		fmt.Println("    mage wsl:build    - Build using WSL explicitly")
		fmt.Println("    mage wsl:up       - Start using WSL explicitly")
		fmt.Println("    mage wsl:down     - Stop using WSL explicitly")
		fmt.Println()
	}
	fmt.Println("  🔧 Utilities:")
	fmt.Println("    mage clean        - Remove containers and volumes")
	fmt.Println()
	fmt.Println("  ℹ️  Info:")
	fmt.Println("    mage -l           - List all targets")
	fmt.Println("    mage help         - Show this help")
	fmt.Println()
	return nil
}

// dockerCompose returns the appropriate docker compose command for the current platform
func dockerCompose(args ...string) error {
	if runtime.GOOS == "windows" {
		// On Windows, use WSL to run docker compose
		projectPath := "/mnt/c/Users/daich/OneDrive/qumo"
		cmdStr := fmt.Sprintf("cd %s && docker compose %s", projectPath, strings.Join(args, " "))
		cmd := exec.Command("wsl", "-d", "Ubuntu", "bash", "-c", cmdStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	// On macOS/Linux, run docker compose directly
	return sh.RunV("docker", append([]string{"compose"}, args...)...)
}

// dockerExec executes a docker command
func dockerExec(args ...string) (string, error) {
	if runtime.GOOS == "windows" {
		cmdStr := fmt.Sprintf("docker %s", strings.Join(args, " "))
		cmd := exec.Command("wsl", "-d", "Ubuntu", "bash", "-c", cmdStr)
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	}
	return sh.Output("docker", args...)
}

// Build builds the Docker image
func Build() error {
	fmt.Println("🔨 Building Docker image...")
	return dockerCompose("build")
}

// Up starts the Docker containers
func Up() error {
	fmt.Println("🚀 Starting Docker containers...")
	return dockerCompose("up", "-d")
}

// Down stops the Docker containers
func Down() error {
	fmt.Println("🛑 Stopping Docker containers...")
	return dockerCompose("down")
}

// Logs shows the container logs
func Logs() error {
	return dockerCompose("logs", "-f")
}

// E2E runs end-to-end tests
func E2E() error {
	mg.Deps(ensureRunning)

	fmt.Println("\n=== E2E Test for qumo-relay ===\n")

	// Test 1: Health Check
	if err := testHealthCheck(); err != nil {
		return err
	}

	// Test 2: Liveness Probe
	if err := testLiveness(); err != nil {
		return err
	}

	// Test 3: Readiness Probe
	if err := testReadiness(); err != nil {
		return err
	}

	// Test 4: Metrics
	if err := testMetrics(); err != nil {
		return err
	}

	// Test 5: Container Status
	if err := testContainerStatus(); err != nil {
		return err
	}

	// Test 6: Port Binding
	if err := testPortBinding(); err != nil {
		return err
	}

	// Summary
	fmt.Println("\n=== E2E Test Summary ===")
	fmt.Println("✅ All tests passed!")
	fmt.Println("\nServer is running at:")
	fmt.Println("  - QUIC Relay: udp://localhost:5000")
	fmt.Println("  - Health Check: http://localhost:8080/health")
	fmt.Println("  - Metrics: http://localhost:8080/metrics")

	return nil
}

// CI runs the full CI pipeline: build, up, test, down
func CI() error {
	fmt.Println("🔄 Running CI pipeline...")

	if err := Build(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if err := Up(); err != nil {
		return fmt.Errorf("up failed: %w", err)
	}

	// Wait for services to be ready
	fmt.Println("⏳ Waiting for services to be ready...")
	time.Sleep(5 * time.Second)

	// Run tests
	testErr := E2E()

	// Always cleanup
	if err := Down(); err != nil {
		fmt.Printf("⚠️  Warning: cleanup failed: %v\n", err)
	}

	if testErr != nil {
		return fmt.Errorf("tests failed: %w", testErr)
	}

	return nil
}

// Helper functions

func ensureRunning() error {
	// Check if container is running
	out, err := dockerExec("ps", "--filter", "name=qumo-relay", "--format", "{{.Status}}")
	if err != nil || !strings.Contains(out, "Up") {
		fmt.Println("⚠️  Container not running, starting it...")
		return Up()
	}
	return nil
}

func wslUp() error {
	fmt.Println("🚀 Starting Docker containers (WSL)...")
	cmd := exec.Command("wsl", "-d", "Ubuntu", "bash", "-c", "cd /mnt/c/Users/daich/OneDrive/qumo && docker compose up -d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func testHealthCheck() error {
	fmt.Println("📋 [TEST 1] Health Check Endpoint")

	out, err := sh.Output("curl", "-s", "http://localhost:8080/health")
	if err != nil {
		return fmt.Errorf("❌ health check failed: %w", err)
	}

	var health struct {
		Status  string `json:"status"`
		Uptime  string `json:"uptime"`
		Version string `json:"version"`
	}

	if err := json.Unmarshal([]byte(out), &health); err != nil {
		return fmt.Errorf("❌ failed to parse health response: %w", err)
	}

	if health.Status != "healthy" {
		return fmt.Errorf("❌ expected status 'healthy', got '%s'", health.Status)
	}

	fmt.Printf("✅ Health check passed\n")
	fmt.Printf("   - Status: %s\n", health.Status)
	fmt.Printf("   - Uptime: %s\n", health.Uptime)
	fmt.Printf("   - Version: %s\n\n", health.Version)

	return nil
}

func testLiveness() error {
	fmt.Println("💓 [TEST 2] Liveness Probe")

	out, err := sh.Output("curl", "-s", "http://localhost:8080/health/live")
	if err != nil {
		return fmt.Errorf("❌ liveness probe failed: %w", err)
	}

	var live struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal([]byte(out), &live); err != nil {
		return fmt.Errorf("❌ failed to parse liveness response: %w", err)
	}

	if live.Status != "alive" {
		return fmt.Errorf("❌ expected status 'alive', got '%s'", live.Status)
	}

	fmt.Println("✅ Liveness probe passed\n")
	return nil
}

func testReadiness() error {
	fmt.Println("🎯 [TEST 3] Readiness Probe")

	out, err := sh.Output("curl", "-s", "http://localhost:8080/health/ready")
	if err != nil {
		return fmt.Errorf("❌ readiness probe failed: %w", err)
	}

	var ready struct {
		Ready bool `json:"ready"`
	}

	if err := json.Unmarshal([]byte(out), &ready); err != nil {
		return fmt.Errorf("❌ failed to parse readiness response: %w", err)
	}

	if !ready.Ready {
		return fmt.Errorf("❌ expected ready=true, got false")
	}

	fmt.Println("✅ Readiness probe passed\n")
	return nil
}

func testMetrics() error {
	fmt.Println("📊 [TEST 4] Metrics Endpoint")

	out, err := sh.Output("curl", "-s", "http://localhost:8080/metrics")
	if err != nil {
		return fmt.Errorf("❌ metrics endpoint failed: %w", err)
	}

	if !strings.Contains(out, "go_goroutines") {
		return fmt.Errorf("❌ metrics do not contain expected data")
	}

	fmt.Println("✅ Metrics endpoint accessible\n")
	return nil
}

func testContainerStatus() error {
	fmt.Println("🐳 [TEST 5] Container Status")

	out, err := dockerExec("ps", "--filter", "name=qumo-relay", "--format", "{{.Status}}")
	if err != nil {
		return fmt.Errorf("❌ failed to check container status: %w", err)
	}

	if !strings.Contains(out, "Up") {
		return fmt.Errorf("❌ container is not running")
	}

	fmt.Printf("✅ Container is running\n")
	fmt.Printf("   - Status: %s\n\n", out)
	return nil
}

func testPortBinding() error {
	fmt.Println("🔌 [TEST 6] Port Binding")

	out, err := dockerExec("ps", "--filter", "name=qumo-relay", "--format", "{{.Ports}}")
	if err != nil {
		return fmt.Errorf("❌ failed to check port binding: %w", err)
	}

	if !strings.Contains(out, "5000/udp") {
		return fmt.Errorf("❌ UDP port 5000 not bound")
	}

	if !strings.Contains(out, "8080/tcp") {
		return fmt.Errorf("❌ TCP port 8080 not bound")
	}

	fmt.Println("✅ Ports are correctly bound")
	fmt.Println("   - UDP 5000: QUIC relay")
	fmt.Println("   - TCP 8080: HTTP health/metrics\n")
	return nil
}

// Clean removes all containers and images
func Clean() error {
	fmt.Println("🧹 Cleaning up...")

	if err := Down(); err != nil {
		return err
	}

	return dockerCompose("down", "--volumes", "--remove-orphans")
}

// Web starts the web demo application
func Web() error {
	mg.Deps(ensureRunning)
	
	fmt.Println("🌐 Starting web demo...")
	fmt.Println("   Relay: http://localhost:8080/health")
	fmt.Println("   Web Demo: http://localhost:5173")
	fmt.Println()
	
	webDir := "web"
	if runtime.GOOS == "windows" {
		webDir = "web"
	}
	
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

// getProjectPath returns the project path based on the OS
func getProjectPath() string {
	if runtime.GOOS == "windows" {
		// On Windows with WSL, use WSL path
		return "/mnt/c/Users/daich/OneDrive/qumo"
	}
	// On macOS/Linux, use current directory
	pwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return pwd
}

// WSL provides WSL-specific commands (Windows only)
type WSL mg.Namespace

// Build builds the Docker image using WSL
func (WSL) Build() error {
	fmt.Println("🔨 Building Docker image (WSL)...")
	cmd := exec.Command("wsl", "-d", "Ubuntu", "bash", "-c", "cd /mnt/c/Users/daich/OneDrive/qumo && docker compose build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Up starts containers using WSL
func (WSL) Up() error {
	fmt.Println("🚀 Starting Docker containers (WSL)...")
	cmd := exec.Command("wsl", "-d", "Ubuntu", "bash", "-c", "cd /mnt/c/Users/daich/OneDrive/qumo && docker compose up -d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Down stops containers using WSL
func (WSL) Down() error {
	fmt.Println("🛑 Stopping Docker containers (WSL)...")
	cmd := exec.Command("wsl", "-d", "Ubuntu", "bash", "-c", "cd /mnt/c/Users/daich/OneDrive/qumo && docker compose down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
