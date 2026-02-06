package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// defaultComposeFiles is the priority order for finding compose files.
var defaultComposeFiles = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yml",
	"docker-compose.yaml",
}

// Load parses compose files and returns a fully resolved ComposeFile.
// If files is empty, it searches projectDir for default compose file names.
// If projectDir is empty, the current working directory is used.
func Load(files []string, projectDir string) (*ComposeFile, error) {
	if projectDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		projectDir = wd
	}

	if len(files) == 0 {
		found, err := findDefaultFile(projectDir)
		if err != nil {
			return nil, err
		}
		files = []string{found}
	}

	var merged *ComposeFile
	for _, f := range files {
		path := f
		if !filepath.IsAbs(path) {
			path = filepath.Join(projectDir, path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}

		data = []byte(interpolateEnv(string(data)))

		cf, err := parseComposeFile(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}

		if merged == nil {
			merged = cf
		} else {
			mergeComposeFiles(merged, cf)
		}
	}

	if merged == nil {
		return nil, fmt.Errorf("no compose files loaded")
	}

	// Resolve flexible types in all services.
	for name, svc := range merged.Services {
		resolved, err := resolveService(svc)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}
		merged.Services[name] = resolved
	}

	return merged, nil
}

// findDefaultFile searches for compose files in priority order.
func findDefaultFile(dir string) (string, error) {
	for _, name := range defaultComposeFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no compose file found in %s (tried: %s)", dir, strings.Join(defaultComposeFiles, ", "))
}

// envVarPattern matches ${VAR}, ${VAR:-default}, and ${VAR-default}.
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// interpolateEnv replaces ${VAR}, ${VAR:-default}, and ${VAR-default} with environment values.
func interpolateEnv(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Strip ${ and }
		inner := match[2 : len(match)-1]

		// Check for ${VAR:-default} (use default if unset or empty)
		if idx := strings.Index(inner, ":-"); idx >= 0 {
			varName := inner[:idx]
			defaultVal := inner[idx+2:]
			if val, ok := os.LookupEnv(varName); ok && val != "" {
				return val
			}
			return defaultVal
		}

		// Check for ${VAR-default} (use default only if unset)
		if idx := strings.Index(inner, "-"); idx >= 0 {
			varName := inner[:idx]
			defaultVal := inner[idx+1:]
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return defaultVal
		}

		// Plain ${VAR}
		return os.Getenv(inner)
	})
}

// parseComposeFile unmarshals YAML data into a ComposeFile.
func parseComposeFile(data []byte) (*ComposeFile, error) {
	var cf ComposeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, err
	}
	if cf.Services == nil {
		cf.Services = make(map[string]Service)
	}
	return &cf, nil
}

// mergeComposeFiles merges src into dst. Services in src override those in dst.
func mergeComposeFiles(dst, src *ComposeFile) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	for name, svc := range src.Services {
		dst.Services[name] = svc
	}
	if dst.Networks == nil && src.Networks != nil {
		dst.Networks = make(map[string]Network)
	}
	for name, net := range src.Networks {
		dst.Networks[name] = net
	}
	if dst.Volumes == nil && src.Volumes != nil {
		dst.Volumes = make(map[string]VolumeConfig)
	}
	for name, vol := range src.Volumes {
		dst.Volumes[name] = vol
	}
}

// resolveService normalizes flexible YAML types in a service definition.
func resolveService(svc Service) (Service, error) {
	var err error

	svc.Command, err = resolveCommand(svc.Command)
	if err != nil {
		return svc, fmt.Errorf("command: %w", err)
	}

	svc.Entrypoint, err = resolveCommand(svc.Entrypoint)
	if err != nil {
		return svc, fmt.Errorf("entrypoint: %w", err)
	}

	svc.Environment, err = resolveEnvironment(svc.Environment)
	if err != nil {
		return svc, fmt.Errorf("environment: %w", err)
	}

	svc.EnvFile, err = resolveEnvFile(svc.EnvFile)
	if err != nil {
		return svc, fmt.Errorf("env_file: %w", err)
	}

	svc.DependsOn, err = resolveDependsOn(svc.DependsOn)
	if err != nil {
		return svc, fmt.Errorf("depends_on: %w", err)
	}

	svc.DNS, err = resolveStringOrList(svc.DNS)
	if err != nil {
		return svc, fmt.Errorf("dns: %w", err)
	}

	svc.DNSSearch, err = resolveStringOrList(svc.DNSSearch)
	if err != nil {
		return svc, fmt.Errorf("dns_search: %w", err)
	}

	svc.Tmpfs, err = resolveStringOrList(svc.Tmpfs)
	if err != nil {
		return svc, fmt.Errorf("tmpfs: %w", err)
	}

	svc.Networks, err = resolveNetworks(svc.Networks)
	if err != nil {
		return svc, fmt.Errorf("networks: %w", err)
	}

	var resolvedBuild interface{}
	resolvedBuild, err = resolveBuild(svc.Build)
	if err != nil {
		return svc, fmt.Errorf("build: %w", err)
	}
	svc.Build = resolvedBuild

	return svc, nil
}

// resolveCommand normalizes command/entrypoint: string → []string, list passes through.
func resolveCommand(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case string:
		return splitCommand(val), nil
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result, nil
	case []string:
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

// splitCommand splits a shell command string into parts.
func splitCommand(s string) []string {
	return strings.Fields(s)
}

// resolveEnvironment normalizes environment: map or list → map[string]string.
func resolveEnvironment(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]string, len(val))
		for k, v := range val {
			if v == nil {
				result[k] = os.Getenv(k)
			} else {
				result[k] = fmt.Sprintf("%v", v)
			}
		}
		return result, nil
	case map[string]string:
		return val, nil
	case []interface{}:
		result := make(map[string]string, len(val))
		for _, item := range val {
			s := fmt.Sprintf("%v", item)
			if k, v, ok := strings.Cut(s, "="); ok {
				result[k] = v
			} else {
				// Variable with no value inherits from host env.
				result[s] = os.Getenv(s)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

// resolveEnvFile normalizes env_file: string, list of strings, or list of objects with path key → []string.
func resolveEnvFile(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case string:
		return []string{val}, nil
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			switch entry := item.(type) {
			case string:
				result = append(result, entry)
			case map[string]interface{}:
				if p, ok := entry["path"]; ok {
					result = append(result, fmt.Sprintf("%v", p))
				}
			}
		}
		return result, nil
	case []string:
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

// resolveDependsOn normalizes depends_on: list → map, map passes through as map[string]DependsOnCondition.
func resolveDependsOn(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case []interface{}:
		result := make(map[string]DependsOnCondition, len(val))
		for _, item := range val {
			name := fmt.Sprintf("%v", item)
			result[name] = DependsOnCondition{Condition: "service_started"}
		}
		return result, nil
	case map[string]interface{}:
		result := make(map[string]DependsOnCondition, len(val))
		for name, cond := range val {
			dc := DependsOnCondition{Condition: "service_started"}
			if condMap, ok := cond.(map[string]interface{}); ok {
				if c, ok := condMap["condition"]; ok {
					dc.Condition = fmt.Sprintf("%v", c)
				}
				if r, ok := condMap["restart"]; ok {
					if rb, ok := r.(bool); ok {
						dc.Restart = rb
					}
				}
			}
			result[name] = dc
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

// resolveStringOrList normalizes dns/dns_search/tmpfs: string → []string, list passes through.
func resolveStringOrList(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case string:
		return []string{val}, nil
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result, nil
	case []string:
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

// resolveNetworks normalizes networks: list → map, map passes through.
func resolveNetworks(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case []interface{}:
		result := make(map[string]interface{}, len(val))
		for _, item := range val {
			result[fmt.Sprintf("%v", item)] = nil
		}
		return result, nil
	case map[string]interface{}:
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

// resolveBuild normalizes build: string (context path) or map → *BuildConfig.
func resolveBuild(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	switch val := v.(type) {
	case *BuildConfig:
		return val, nil
	case string:
		return &BuildConfig{Context: val}, nil
	case map[string]interface{}:
		bc := &BuildConfig{}
		if c, ok := val["context"]; ok {
			bc.Context = fmt.Sprintf("%v", c)
		}
		if d, ok := val["dockerfile"]; ok {
			bc.Dockerfile = fmt.Sprintf("%v", d)
		}
		if t, ok := val["target"]; ok {
			bc.Target = fmt.Sprintf("%v", t)
		}
		if a, ok := val["args"]; ok {
			if argsMap, ok := a.(map[string]interface{}); ok {
				bc.Args = make(map[string]string, len(argsMap))
				for k, v := range argsMap {
					bc.Args[k] = fmt.Sprintf("%v", v)
				}
			}
		}
		if l, ok := val["labels"]; ok {
			if labelsMap, ok := l.(map[string]interface{}); ok {
				bc.Labels = make(map[string]string, len(labelsMap))
				for k, v := range labelsMap {
					bc.Labels[k] = fmt.Sprintf("%v", v)
				}
			}
		}
		return bc, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}
