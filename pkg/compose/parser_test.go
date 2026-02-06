package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_BasicFile(t *testing.T) {
	dir := t.TempDir()
	content := `
services:
  web:
    image: nginx:latest
    command: "echo hello world"
`
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}

	cf, err := Load(nil, dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	svc, ok := cf.Services["web"]
	if !ok {
		t.Fatal("expected service 'web' to exist")
	}
	if svc.Image != "nginx:latest" {
		t.Errorf("image = %q, want %q", svc.Image, "nginx:latest")
	}

	cmd, ok := svc.Command.([]string)
	if !ok {
		t.Fatalf("command type = %T, want []string", svc.Command)
	}
	wantCmd := []string{"echo", "hello", "world"}
	if len(cmd) != len(wantCmd) {
		t.Fatalf("command len = %d, want %d", len(cmd), len(wantCmd))
	}
	for i := range wantCmd {
		if cmd[i] != wantCmd[i] {
			t.Errorf("command[%d] = %q, want %q", i, cmd[i], wantCmd[i])
		}
	}
}

func TestLoad_EnvironmentInterpolation(t *testing.T) {
	t.Run("plain variable", func(t *testing.T) {
		t.Setenv("TEST_IMAGE_TAG", "v2.0")
		dir := t.TempDir()
		content := `
services:
  app:
    image: myapp:${TEST_IMAGE_TAG}
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cf.Services["app"].Image != "myapp:v2.0" {
			t.Errorf("image = %q, want %q", cf.Services["app"].Image, "myapp:v2.0")
		}
	})

	t.Run("default when unset (:-)", func(t *testing.T) {
		// Ensure the variable is not set.
		os.Unsetenv("TEST_UNSET_VAR_COLON")
		dir := t.TempDir()
		content := `
services:
  app:
    image: myapp:${TEST_UNSET_VAR_COLON:-fallback}
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cf.Services["app"].Image != "myapp:fallback" {
			t.Errorf("image = %q, want %q", cf.Services["app"].Image, "myapp:fallback")
		}
	})

	t.Run("default when empty (:-)", func(t *testing.T) {
		t.Setenv("TEST_EMPTY_VAR_COLON", "")
		dir := t.TempDir()
		content := `
services:
  app:
    image: myapp:${TEST_EMPTY_VAR_COLON:-fallback}
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cf.Services["app"].Image != "myapp:fallback" {
			t.Errorf("image = %q, want %q", cf.Services["app"].Image, "myapp:fallback")
		}
	})

	t.Run(":- uses value when set and non-empty", func(t *testing.T) {
		t.Setenv("TEST_SET_VAR_COLON", "realvalue")
		dir := t.TempDir()
		content := `
services:
  app:
    image: myapp:${TEST_SET_VAR_COLON:-fallback}
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cf.Services["app"].Image != "myapp:realvalue" {
			t.Errorf("image = %q, want %q", cf.Services["app"].Image, "myapp:realvalue")
		}
	})

	t.Run("default only when unset (-)", func(t *testing.T) {
		os.Unsetenv("TEST_UNSET_VAR_DASH")
		dir := t.TempDir()
		content := `
services:
  app:
    image: myapp:${TEST_UNSET_VAR_DASH-fallback}
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cf.Services["app"].Image != "myapp:fallback" {
			t.Errorf("image = %q, want %q", cf.Services["app"].Image, "myapp:fallback")
		}
	})

	t.Run("- keeps empty value when set to empty", func(t *testing.T) {
		t.Setenv("TEST_EMPTY_VAR_DASH", "")
		dir := t.TempDir()
		content := `
services:
  app:
    image: "myapp:${TEST_EMPTY_VAR_DASH-fallback}"
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		// With plain -, set-but-empty returns the empty string (not the default).
		if cf.Services["app"].Image != "myapp:" {
			t.Errorf("image = %q, want %q", cf.Services["app"].Image, "myapp:")
		}
	})
}

func TestLoad_CommandFormats(t *testing.T) {
	t.Run("string command", func(t *testing.T) {
		dir := t.TempDir()
		content := `
services:
  app:
    image: alpine
    command: "echo hello world"
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		cmd, ok := cf.Services["app"].Command.([]string)
		if !ok {
			t.Fatalf("command type = %T, want []string", cf.Services["app"].Command)
		}
		want := []string{"echo", "hello", "world"}
		if len(cmd) != len(want) {
			t.Fatalf("command len = %d, want %d", len(cmd), len(want))
		}
		for i := range want {
			if cmd[i] != want[i] {
				t.Errorf("command[%d] = %q, want %q", i, cmd[i], want[i])
			}
		}
	})

	t.Run("list command", func(t *testing.T) {
		dir := t.TempDir()
		content := `
services:
  app:
    image: alpine
    command: ["echo", "hello"]
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		cmd, ok := cf.Services["app"].Command.([]string)
		if !ok {
			t.Fatalf("command type = %T, want []string", cf.Services["app"].Command)
		}
		want := []string{"echo", "hello"}
		if len(cmd) != len(want) {
			t.Fatalf("command len = %d, want %d", len(cmd), len(want))
		}
		for i := range want {
			if cmd[i] != want[i] {
				t.Errorf("command[%d] = %q, want %q", i, cmd[i], want[i])
			}
		}
	})
}

func TestLoad_DependsOnFormats(t *testing.T) {
	t.Run("list format", func(t *testing.T) {
		dir := t.TempDir()
		content := `
services:
  web:
    image: nginx
    depends_on:
      - db
      - redis
  db:
    image: postgres
  redis:
    image: redis
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		deps, ok := cf.Services["web"].DependsOn.(map[string]DependsOnCondition)
		if !ok {
			t.Fatalf("depends_on type = %T, want map[string]DependsOnCondition", cf.Services["web"].DependsOn)
		}
		if len(deps) != 2 {
			t.Fatalf("depends_on len = %d, want 2", len(deps))
		}
		for _, name := range []string{"db", "redis"} {
			dc, ok := deps[name]
			if !ok {
				t.Errorf("expected depends_on to contain %q", name)
				continue
			}
			if dc.Condition != "service_started" {
				t.Errorf("depends_on[%q].Condition = %q, want %q", name, dc.Condition, "service_started")
			}
		}
	})

	t.Run("map format with conditions", func(t *testing.T) {
		dir := t.TempDir()
		content := `
services:
  web:
    image: nginx
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_started
  db:
    image: postgres
  redis:
    image: redis
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		deps, ok := cf.Services["web"].DependsOn.(map[string]DependsOnCondition)
		if !ok {
			t.Fatalf("depends_on type = %T, want map[string]DependsOnCondition", cf.Services["web"].DependsOn)
		}
		if deps["db"].Condition != "service_healthy" {
			t.Errorf("depends_on[db].Condition = %q, want %q", deps["db"].Condition, "service_healthy")
		}
		if deps["redis"].Condition != "service_started" {
			t.Errorf("depends_on[redis].Condition = %q, want %q", deps["redis"].Condition, "service_started")
		}
	})
}

func TestLoad_BuildFormats(t *testing.T) {
	t.Run("string format", func(t *testing.T) {
		dir := t.TempDir()
		content := `
services:
  app:
    build: ./app
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		bc, ok := cf.Services["app"].Build.(*BuildConfig)
		if !ok {
			t.Fatalf("build type = %T, want *BuildConfig", cf.Services["app"].Build)
		}
		if bc.Context != "./app" {
			t.Errorf("build.Context = %q, want %q", bc.Context, "./app")
		}
	})

	t.Run("full config", func(t *testing.T) {
		dir := t.TempDir()
		content := `
services:
  app:
    build:
      context: ./app
      dockerfile: Dockerfile.prod
      args:
        ENV: production
      target: builder
      labels:
        version: "1.0"
`
		if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("writing compose file: %v", err)
		}
		cf, err := Load(nil, dir)
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		bc, ok := cf.Services["app"].Build.(*BuildConfig)
		if !ok {
			t.Fatalf("build type = %T, want *BuildConfig", cf.Services["app"].Build)
		}
		if bc.Context != "./app" {
			t.Errorf("build.Context = %q, want %q", bc.Context, "./app")
		}
		if bc.Dockerfile != "Dockerfile.prod" {
			t.Errorf("build.Dockerfile = %q, want %q", bc.Dockerfile, "Dockerfile.prod")
		}
		if bc.Target != "builder" {
			t.Errorf("build.Target = %q, want %q", bc.Target, "builder")
		}
		if bc.Args["ENV"] != "production" {
			t.Errorf("build.Args[ENV] = %q, want %q", bc.Args["ENV"], "production")
		}
		if bc.Labels["version"] != "1.0" {
			t.Errorf("build.Labels[version] = %q, want %q", bc.Labels["version"], "1.0")
		}
	})
}

func TestLoad_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	base := `
services:
  web:
    image: nginx:1.25
    ports:
      - "80:80"
  db:
    image: postgres:15
`
	override := `
services:
  web:
    image: nginx:1.26
    ports:
      - "8080:80"
  cache:
    image: redis:7
`
	basePath := filepath.Join(dir, "compose.yaml")
	overridePath := filepath.Join(dir, "compose.override.yaml")
	if err := os.WriteFile(basePath, []byte(base), 0o644); err != nil {
		t.Fatalf("writing base compose file: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte(override), 0o644); err != nil {
		t.Fatalf("writing override compose file: %v", err)
	}

	cf, err := Load([]string{basePath, overridePath}, dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// web should be overridden by the second file.
	if cf.Services["web"].Image != "nginx:1.26" {
		t.Errorf("web.Image = %q, want %q", cf.Services["web"].Image, "nginx:1.26")
	}
	// db should survive from the base file.
	if cf.Services["db"].Image != "postgres:15" {
		t.Errorf("db.Image = %q, want %q", cf.Services["db"].Image, "postgres:15")
	}
	// cache should be added from the override.
	if cf.Services["cache"].Image != "redis:7" {
		t.Errorf("cache.Image = %q, want %q", cf.Services["cache"].Image, "redis:7")
	}
}

func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(nil, dir)
	if err == nil {
		t.Fatal("expected error when no compose file exists")
	}
	if got := err.Error(); !strings.Contains(got, "no compose file found") {
		t.Errorf("error = %q, want it to contain %q", got, "no compose file found")
	}
}

func TestLoad_DefaultFileDiscovery(t *testing.T) {
	// Verify each default file name is found in priority order.
	for _, name := range []string{"compose.yaml", "compose.yml", "docker-compose.yml", "docker-compose.yaml"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			content := `
services:
  app:
    image: alpine
`
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				t.Fatalf("writing %s: %v", name, err)
			}
			cf, err := Load(nil, dir)
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}
			if _, ok := cf.Services["app"]; !ok {
				t.Error("expected service 'app' to exist")
			}
		})
	}
}

func TestLoad_RelativeFilePath(t *testing.T) {
	dir := t.TempDir()
	content := `
services:
  app:
    image: alpine
`
	if err := os.WriteFile(filepath.Join(dir, "custom.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}

	cf, err := Load([]string{"custom.yaml"}, dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if _, ok := cf.Services["app"]; !ok {
		t.Error("expected service 'app' to exist")
	}
}

func TestLoad_EnvironmentMapFormat(t *testing.T) {
	dir := t.TempDir()
	content := `
services:
  app:
    image: alpine
    environment:
      FOO: bar
      NUM: "42"
`
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}
	cf, err := Load(nil, dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	env, ok := cf.Services["app"].Environment.(map[string]string)
	if !ok {
		t.Fatalf("environment type = %T, want map[string]string", cf.Services["app"].Environment)
	}
	if env["FOO"] != "bar" {
		t.Errorf("env[FOO] = %q, want %q", env["FOO"], "bar")
	}
	if env["NUM"] != "42" {
		t.Errorf("env[NUM] = %q, want %q", env["NUM"], "42")
	}
}

func TestLoad_EnvironmentListFormat(t *testing.T) {
	dir := t.TempDir()
	content := `
services:
  app:
    image: alpine
    environment:
      - FOO=bar
      - BAZ=qux
`
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}
	cf, err := Load(nil, dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	env, ok := cf.Services["app"].Environment.(map[string]string)
	if !ok {
		t.Fatalf("environment type = %T, want map[string]string", cf.Services["app"].Environment)
	}
	if env["FOO"] != "bar" {
		t.Errorf("env[FOO] = %q, want %q", env["FOO"], "bar")
	}
	if env["BAZ"] != "qux" {
		t.Errorf("env[BAZ] = %q, want %q", env["BAZ"], "qux")
	}
}

func TestLoad_ProjectName(t *testing.T) {
	dir := t.TempDir()
	content := `
name: myproject
services:
  app:
    image: alpine
`
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}
	cf, err := Load(nil, dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cf.Name != "myproject" {
		t.Errorf("Name = %q, want %q", cf.Name, "myproject")
	}
}

func TestLoad_Networks(t *testing.T) {
	dir := t.TempDir()
	content := `
services:
  app:
    image: alpine
networks:
  frontend:
    driver: bridge
  backend:
    internal: true
`
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}
	cf, err := Load(nil, dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cf.Networks) != 2 {
		t.Fatalf("networks len = %d, want 2", len(cf.Networks))
	}
	if cf.Networks["frontend"].Driver != "bridge" {
		t.Errorf("networks[frontend].Driver = %q, want %q", cf.Networks["frontend"].Driver, "bridge")
	}
	if !cf.Networks["backend"].Internal {
		t.Error("expected networks[backend].Internal to be true")
	}
}

func TestLoad_Volumes(t *testing.T) {
	dir := t.TempDir()
	content := `
services:
  app:
    image: alpine
volumes:
  data:
    driver: local
`
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing compose file: %v", err)
	}
	cf, err := Load(nil, dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cf.Volumes) != 1 {
		t.Fatalf("volumes len = %d, want 1", len(cf.Volumes))
	}
	if cf.Volumes["data"].Driver != "local" {
		t.Errorf("volumes[data].Driver = %q, want %q", cf.Volumes["data"].Driver, "local")
	}
}

