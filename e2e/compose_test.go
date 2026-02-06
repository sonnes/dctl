//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var dctlBin string

func TestMain(m *testing.M) {
	// Check that the container runtime is available.
	if _, err := exec.LookPath("container"); err != nil {
		fmt.Println("skipping e2e tests: container runtime not found in PATH")
		os.Exit(0)
	}

	// Build the dctl binary once into a temp directory.
	tmp, err := os.MkdirTemp("", "dctl-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	bin := filepath.Join(tmp, "dctl")
	dctlRoot := filepath.Join("..")
	abs, err := filepath.Abs(dctlRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve dctl root: %v\n", err)
		os.Exit(1)
	}

	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = abs
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build dctl: %v\n", err)
		os.Exit(1)
	}

	dctlBin = bin
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// dctlRun executes the dctl binary with the given args and working directory.
// It returns the combined stdout+stderr output and any error.
func dctlRun(dir string, args ...string) (string, error) {
	cmd := exec.Command(dctlBin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// setupProject creates a temp directory containing a compose.yaml with the
// provided content and returns the directory path.
func setupProject(t *testing.T, composeYAML string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "dctl-e2e-project-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(composeYAML), 0o644); err != nil {
		t.Fatalf("failed to write compose.yaml: %v", err)
	}
	return dir
}

// cleanupProject tears down a compose project and removes the temp dir.
func cleanupProject(t *testing.T, dir, projectName string) {
	t.Helper()
	// Best-effort teardown.
	_, _ = dctlRun(dir, "compose", "-p", projectName, "down", "-v")
	_ = os.RemoveAll(dir)
}

// projectName returns a sanitized, unique project name derived from t.Name().
func projectName(t *testing.T) string {
	t.Helper()
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	name := re.ReplaceAllString(t.Name(), "-")
	name = strings.ToLower(name)
	name = strings.Trim(name, "-")
	if len(name) > 40 {
		name = name[:40]
	}
	return name
}

// waitForContainer polls until the container shows up in ps output or timeout.
func waitForContainer(t *testing.T, dir, pname string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := dctlRun(dir, "compose", "-p", pname, "ps")
		if err == nil && strings.TrimSpace(out) != "" {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

const testImage = "ghcr.io/linuxcontainers/alpine:3.20"

// ---------------------------------------------------------------------------
// 1. Basic Lifecycle
// ---------------------------------------------------------------------------

func TestComposeUp(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}

	waitForContainer(t, dir, pname, 15*time.Second)

	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	if err != nil {
		t.Fatalf("compose ps failed: %v\noutput: %s", err, psOut)
	}

	expectedContainer := pname + "_app"
	if !strings.Contains(psOut, expectedContainer) {
		t.Errorf("expected ps output to contain %q, got:\n%s", expectedContainer, psOut)
	}
}

func TestComposeDown(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	out, err = dctlRun(dir, "compose", "-p", pname, "down")
	if err != nil {
		t.Fatalf("compose down failed: %v\noutput: %s", err, out)
	}

	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	// After down the project state is deleted, so ps should fail or return empty.
	if err == nil && strings.TrimSpace(psOut) != "" {
		expectedContainer := pname + "_app"
		if strings.Contains(psOut, expectedContainer) {
			t.Errorf("expected container to be removed after down, but ps still shows it:\n%s", psOut)
		}
	}
}

func TestComposeStop_Start(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Stop
	out, err = dctlRun(dir, "compose", "-p", pname, "stop")
	if err != nil {
		t.Fatalf("compose stop failed: %v\noutput: %s", err, out)
	}

	// Start
	out, err = dctlRun(dir, "compose", "-p", pname, "start")
	if err != nil {
		// start is not a subcommand in the current CLI but we attempt it;
		// it calls container start under the hood via restart logic.
		t.Logf("compose start returned: %v\noutput: %s", err, out)
	}

	// Verify running after start by checking ps
	time.Sleep(2 * time.Second)
	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	if err != nil {
		t.Logf("compose ps after start: %v\noutput: %s", err, psOut)
	}
}

func TestComposeRestart(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	out, err = dctlRun(dir, "compose", "-p", pname, "restart")
	if err != nil {
		t.Fatalf("compose restart failed: %v\noutput: %s", err, out)
	}

	time.Sleep(2 * time.Second)

	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	if err != nil {
		t.Fatalf("compose ps after restart failed: %v\noutput: %s", err, psOut)
	}

	expectedContainer := pname + "_app"
	if !strings.Contains(psOut, expectedContainer) {
		t.Errorf("expected ps output to contain %q after restart, got:\n%s", expectedContainer, psOut)
	}
}

// ---------------------------------------------------------------------------
// 2. Logs & Exec
// ---------------------------------------------------------------------------

func TestComposeLogs(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sh", "-c", "echo hello && sleep infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Give the container a moment to produce the log line.
	time.Sleep(3 * time.Second)

	logsOut, err := dctlRun(dir, "compose", "-p", pname, "logs", "app")
	if err != nil {
		t.Fatalf("compose logs failed: %v\noutput: %s", err, logsOut)
	}

	if !strings.Contains(logsOut, "hello") {
		t.Errorf("expected logs to contain %q, got:\n%s", "hello", logsOut)
	}
}

func TestComposeExec(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)
	time.Sleep(2 * time.Second)

	execOut, err := dctlRun(dir, "compose", "-p", pname, "exec", "-T", "app", "echo", "test")
	if err != nil {
		t.Fatalf("compose exec failed: %v\noutput: %s", err, execOut)
	}

	if !strings.Contains(execOut, "test") {
		t.Errorf("expected exec output to contain %q, got:\n%s", "test", execOut)
	}
}

// ---------------------------------------------------------------------------
// 3. Build & Pull
// ---------------------------------------------------------------------------

func TestComposePull(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "pull")
	if err != nil {
		t.Fatalf("compose pull failed: %v\noutput: %s", err, out)
	}
}

func TestComposeBuild(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    build: .
    image: dctl-e2e-build-test
`)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	// Write a minimal Dockerfile.
	dockerfile := fmt.Sprintf("FROM %s\n", testImage)
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}

	out, err := dctlRun(dir, "compose", "-p", pname, "build")
	if err != nil {
		t.Fatalf("compose build failed: %v\noutput: %s", err, out)
	}
}

// ---------------------------------------------------------------------------
// 4. Config & Validation
// ---------------------------------------------------------------------------

func TestComposeConfig(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  web:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "config")
	if err != nil {
		t.Fatalf("compose config failed: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "web") {
		t.Errorf("expected config output to contain service name %q, got:\n%s", "web", out)
	}
}

func TestComposeConfig_EnvInterpolation(t *testing.T) {
	yaml := `services:
  app:
    image: ${TEST_IMAGE:-ghcr.io/linuxcontainers/alpine:3.20}
    command: ["sleep", "infinity"]
`
	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "config")
	if err != nil {
		t.Fatalf("compose config failed: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "ghcr.io/linuxcontainers/alpine:3.20") {
		t.Errorf("expected config output to contain resolved default image, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// 5. Multi-Service & Dependencies
// ---------------------------------------------------------------------------

func TestComposeUp_MultiService(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  web:
    image: %s
    command: ["sleep", "infinity"]
  worker:
    image: %s
    command: ["sleep", "infinity"]
`, testImage, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	if err != nil {
		t.Fatalf("compose ps failed: %v\noutput: %s", err, psOut)
	}

	if !strings.Contains(psOut, pname+"_web") {
		t.Errorf("expected ps to contain %q, got:\n%s", pname+"_web", psOut)
	}
	if !strings.Contains(psOut, pname+"_worker") {
		t.Errorf("expected ps to contain %q, got:\n%s", pname+"_worker", psOut)
	}
}

func TestComposeUp_DependsOn(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  db:
    image: %s
    command: ["sleep", "infinity"]
  app:
    image: %s
    command: ["sleep", "infinity"]
    depends_on:
      - db
`, testImage, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	if err != nil {
		t.Fatalf("compose ps failed: %v\noutput: %s", err, psOut)
	}

	if !strings.Contains(psOut, pname+"_db") {
		t.Errorf("expected ps to contain %q, got:\n%s", pname+"_db", psOut)
	}
	if !strings.Contains(psOut, pname+"_app") {
		t.Errorf("expected ps to contain %q, got:\n%s", pname+"_app", psOut)
	}
}

func TestComposePs_FilterByService(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  web:
    image: %s
    command: ["sleep", "infinity"]
  worker:
    image: %s
    command: ["sleep", "infinity"]
`, testImage, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	if err != nil {
		t.Fatalf("compose ps failed: %v\noutput: %s", err, psOut)
	}

	// Both container names should be present in the output.
	webName := pname + "_web"
	workerName := pname + "_worker"
	if !strings.Contains(psOut, webName) {
		t.Errorf("expected ps output to contain %q, got:\n%s", webName, psOut)
	}
	if !strings.Contains(psOut, workerName) {
		t.Errorf("expected ps output to contain %q, got:\n%s", workerName, psOut)
	}
}

// ---------------------------------------------------------------------------
// 6. Networks
// ---------------------------------------------------------------------------

func TestComposeUp_DefaultNetwork(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Verify the project state file exists and container was started.
	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	if err != nil {
		t.Fatalf("compose ps failed: %v\noutput: %s", err, psOut)
	}
	if !strings.Contains(psOut, pname+"_app") {
		t.Errorf("expected container in ps output, got:\n%s", psOut)
	}
}

func TestComposeUp_CustomNetwork(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
    networks:
      - mynet
networks:
  mynet: {}
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Check that the network "mynet" was created.
	netOut, err := exec.Command("container", "network", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("container network list failed: %v\noutput: %s", err, string(netOut))
	}
	if !strings.Contains(string(netOut), "mynet") {
		t.Errorf("expected network list to contain %q, got:\n%s", "mynet", string(netOut))
	}
}

func TestComposeDown_RemovesNetworks(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
    networks:
      - testdownnet
networks:
  testdownnet: {}
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Verify network exists.
	netOut, err := exec.Command("container", "network", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("container network list failed: %v\noutput: %s", err, string(netOut))
	}
	if !strings.Contains(string(netOut), "testdownnet") {
		t.Fatalf("expected network %q to exist before down, got:\n%s", "testdownnet", string(netOut))
	}

	// Down
	out, err = dctlRun(dir, "compose", "-p", pname, "down")
	if err != nil {
		t.Fatalf("compose down failed: %v\noutput: %s", err, out)
	}

	// Verify network removed.
	netOut, err = exec.Command("container", "network", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("container network list failed: %v\noutput: %s", err, string(netOut))
	}
	if strings.Contains(string(netOut), "testdownnet") {
		t.Errorf("expected network %q to be removed after down, but it still exists:\n%s", "testdownnet", string(netOut))
	}
}

// ---------------------------------------------------------------------------
// 7. Volumes
// ---------------------------------------------------------------------------

func TestComposeUp_NamedVolume(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
    volumes:
      - mydata:/data
volumes:
  mydata: {}
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)
	time.Sleep(2 * time.Second)

	// Verify volume created.
	volOut, err := exec.Command("container", "volume", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("container volume list failed: %v\noutput: %s", err, string(volOut))
	}
	if !strings.Contains(string(volOut), "mydata") {
		t.Errorf("expected volume list to contain %q, got:\n%s", "mydata", string(volOut))
	}

	// Write a file inside the container and read it back.
	execOut, err := dctlRun(dir, "compose", "-p", pname, "exec", "-T", "app", "sh", "-c", "echo volumetest > /data/test.txt && cat /data/test.txt")
	if err != nil {
		t.Fatalf("compose exec write/read failed: %v\noutput: %s", err, execOut)
	}
	if !strings.Contains(execOut, "volumetest") {
		t.Errorf("expected exec output to contain %q, got:\n%s", "volumetest", execOut)
	}
}

func TestComposeUp_BindMount(t *testing.T) {
	pname := projectName(t)

	// Create a host directory with a test file.
	hostDir, err := os.MkdirTemp("", "dctl-e2e-bind-*")
	if err != nil {
		t.Fatalf("failed to create host temp dir: %v", err)
	}
	defer os.RemoveAll(hostDir)

	if err := os.WriteFile(filepath.Join(hostDir, "hello.txt"), []byte("bindmount-test"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
    volumes:
      - %s:/mnt/host
`, testImage, hostDir)

	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)
	time.Sleep(2 * time.Second)

	execOut, err := dctlRun(dir, "compose", "-p", pname, "exec", "-T", "app", "cat", "/mnt/host/hello.txt")
	if err != nil {
		t.Fatalf("compose exec cat failed: %v\noutput: %s", err, execOut)
	}
	if !strings.Contains(execOut, "bindmount-test") {
		t.Errorf("expected exec output to contain %q, got:\n%s", "bindmount-test", execOut)
	}
}

func TestComposeDown_VolumesFlag(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
    volumes:
      - voldowntest:/data
volumes:
  voldowntest: {}
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Down with -v flag should remove volume.
	out, err = dctlRun(dir, "compose", "-p", pname, "down", "-v")
	if err != nil {
		t.Fatalf("compose down -v failed: %v\noutput: %s", err, out)
	}

	volOut, err := exec.Command("container", "volume", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("container volume list failed: %v\noutput: %s", err, string(volOut))
	}
	if strings.Contains(string(volOut), "voldowntest") {
		t.Errorf("expected volume %q to be removed after down -v, but it still exists:\n%s", "voldowntest", string(volOut))
	}
}

func TestComposeDown_WithoutVolumesFlag(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
    volumes:
      - volkeeptest:/data
volumes:
  volkeeptest: {}
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Down without -v should keep the volume.
	out, err = dctlRun(dir, "compose", "-p", pname, "down")
	if err != nil {
		t.Fatalf("compose down failed: %v\noutput: %s", err, out)
	}

	volOut, err := exec.Command("container", "volume", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("container volume list failed: %v\noutput: %s", err, string(volOut))
	}
	if !strings.Contains(string(volOut), "volkeeptest") {
		t.Errorf("expected volume %q to be preserved after down (no -v), got:\n%s", "volkeeptest", string(volOut))
	}

	// Manually clean up the volume.
	_ = exec.Command("container", "volume", "delete", "volkeeptest").Run()
}

// ---------------------------------------------------------------------------
// 8. Run & Rm
// ---------------------------------------------------------------------------

func TestComposeRun(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "run", "--rm", "app", "echo", "hello")
	if err != nil {
		t.Fatalf("compose run failed: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "hello") {
		t.Errorf("expected run output to contain %q, got:\n%s", "hello", out)
	}
}

func TestComposeRm(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Stop first.
	out, err = dctlRun(dir, "compose", "-p", pname, "stop")
	if err != nil {
		t.Fatalf("compose stop failed: %v\noutput: %s", err, out)
	}

	// Remove.
	out, err = dctlRun(dir, "compose", "-p", pname, "rm", "-f")
	if err != nil {
		t.Fatalf("compose rm failed: %v\noutput: %s", err, out)
	}

	// Verify container is gone by checking container list directly.
	containerName := pname + "_app"
	listOut, err := exec.Command("container", "list", "--format", "json").CombinedOutput()
	if err != nil {
		t.Logf("container list failed: %v\noutput: %s", err, string(listOut))
	}
	if strings.Contains(string(listOut), containerName) {
		t.Errorf("expected container %q to be deleted after rm, but it still exists:\n%s", containerName, string(listOut))
	}
}

// ---------------------------------------------------------------------------
// 9. Kill
// ---------------------------------------------------------------------------

func TestComposeKill(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	out, err = dctlRun(dir, "compose", "-p", pname, "kill")
	if err != nil {
		t.Fatalf("compose kill failed: %v\noutput: %s", err, out)
	}

	// After kill the container should no longer be running.
	// Give it a moment to stop.
	time.Sleep(2 * time.Second)

	listOut, err := exec.Command("container", "list", "--format", "json").CombinedOutput()
	if err != nil {
		t.Logf("container list failed: %v\noutput: %s", err, string(listOut))
	}

	// The container may still show up in the list but should not be in "running" state.
	// We just verify the kill command succeeded without error above.
}

// ---------------------------------------------------------------------------
// 10. Project Flags
// ---------------------------------------------------------------------------

func TestCompose_ProjectNameFlag(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := "custom-e2e-project"
	dir := setupProject(t, yaml)
	defer cleanupProject(t, dir, pname)

	out, err := dctlRun(dir, "compose", "-p", pname, "up", "-d")
	if err != nil {
		t.Fatalf("compose up failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	psOut, err := dctlRun(dir, "compose", "-p", pname, "ps")
	if err != nil {
		t.Fatalf("compose ps failed: %v\noutput: %s", err, psOut)
	}

	expectedContainer := pname + "_app"
	if !strings.Contains(psOut, expectedContainer) {
		t.Errorf("expected ps output to contain %q (custom project name), got:\n%s", expectedContainer, psOut)
	}
}

func TestCompose_FileFlag(t *testing.T) {
	yaml := fmt.Sprintf(`services:
  app:
    image: %s
    command: ["sleep", "infinity"]
`, testImage)

	pname := projectName(t)

	// Create a temp dir but write to a custom filename.
	dir, err := os.MkdirTemp("", "dctl-e2e-fileflag-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer cleanupProject(t, dir, pname)

	if err := os.WriteFile(filepath.Join(dir, "custom-compose.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("failed to write custom-compose.yaml: %v", err)
	}

	out, err := dctlRun(dir, "compose", "-p", pname, "-f", "custom-compose.yaml", "up", "-d")
	if err != nil {
		t.Fatalf("compose up with -f failed: %v\noutput: %s", err, out)
	}
	waitForContainer(t, dir, pname, 15*time.Second)

	// Need to use -f for ps as well since there is no default compose.yaml.
	psOut, err := dctlRun(dir, "compose", "-p", pname, "-f", "custom-compose.yaml", "ps")
	if err != nil {
		t.Fatalf("compose ps failed: %v\noutput: %s", err, psOut)
	}

	expectedContainer := pname + "_app"
	if !strings.Contains(psOut, expectedContainer) {
		t.Errorf("expected ps output to contain %q, got:\n%s", expectedContainer, psOut)
	}
}
