package compose

// ComposeFile represents a parsed docker-compose.yml / compose.yaml file.
type ComposeFile struct {
	Name     string                  `yaml:"name,omitempty"`
	Services map[string]Service      `yaml:"services"`
	Networks map[string]Network      `yaml:"networks,omitempty"`
	Volumes  map[string]VolumeConfig `yaml:"volumes,omitempty"`
}

// Service represents a single service definition.
type Service struct {
	Image       string            `yaml:"image,omitempty"`
	Build       interface{}       `yaml:"build,omitempty"`
	Command     interface{}       `yaml:"command,omitempty"`
	Entrypoint  interface{}       `yaml:"entrypoint,omitempty"`
	Environment interface{}       `yaml:"environment,omitempty"`
	EnvFile     interface{}       `yaml:"env_file,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Networks    interface{}       `yaml:"networks,omitempty"`
	DependsOn   interface{}       `yaml:"depends_on,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	User        string            `yaml:"user,omitempty"`
	Hostname    string            `yaml:"hostname,omitempty"`
	DNS         interface{}       `yaml:"dns,omitempty"`
	DNSSearch   interface{}       `yaml:"dns_search,omitempty"`
	ExtraHosts  []string          `yaml:"extra_hosts,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	StdinOpen   bool              `yaml:"stdin_open,omitempty"`
	Tty         bool              `yaml:"tty,omitempty"`
	ReadOnly    bool              `yaml:"read_only,omitempty"`
	Privileged  bool              `yaml:"privileged,omitempty"`
	Init        bool              `yaml:"init,omitempty"`
	Platform    string            `yaml:"platform,omitempty"`
	CPUs        interface{}       `yaml:"cpus,omitempty"`
	MemLimit    string            `yaml:"mem_limit,omitempty"`
	Tmpfs       interface{}       `yaml:"tmpfs,omitempty"`
	Healthcheck *Healthcheck      `yaml:"healthcheck,omitempty"`
	ContainerName string          `yaml:"container_name,omitempty"`
	PullPolicy  string            `yaml:"pull_policy,omitempty"`
	StopSignal  string            `yaml:"stop_signal,omitempty"`
	StopGracePeriod string        `yaml:"stop_grace_period,omitempty"`
}

// BuildConfig represents the build configuration for a service.
type BuildConfig struct {
	Context    string            `yaml:"context,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty"`
	Args       map[string]string `yaml:"args,omitempty"`
	Target     string            `yaml:"target,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
}

// Network represents a network definition.
type Network struct {
	Driver   string            `yaml:"driver,omitempty"`
	Internal bool              `yaml:"internal,omitempty"`
	External bool              `yaml:"external,omitempty"`
	Name     string            `yaml:"name,omitempty"`
	Labels   map[string]string `yaml:"labels,omitempty"`
	IPAM     *IPAM             `yaml:"ipam,omitempty"`
}

// IPAM represents IPAM configuration.
type IPAM struct {
	Driver string       `yaml:"driver,omitempty"`
	Config []IPAMConfig `yaml:"config,omitempty"`
}

// IPAMConfig represents IPAM config.
type IPAMConfig struct {
	Subnet string `yaml:"subnet,omitempty"`
}

// VolumeConfig represents a volume definition.
type VolumeConfig struct {
	Driver   string            `yaml:"driver,omitempty"`
	External bool              `yaml:"external,omitempty"`
	Name     string            `yaml:"name,omitempty"`
	Labels   map[string]string `yaml:"labels,omitempty"`
}

// Healthcheck represents a healthcheck configuration.
type Healthcheck struct {
	Test     interface{} `yaml:"test,omitempty"`
	Interval string      `yaml:"interval,omitempty"`
	Timeout  string      `yaml:"timeout,omitempty"`
	Retries  int         `yaml:"retries,omitempty"`
	Disable  bool        `yaml:"disable,omitempty"`
}

// DependsOnCondition represents a depends_on condition.
type DependsOnCondition struct {
	Condition string `yaml:"condition,omitempty"`
	Restart   bool   `yaml:"restart,omitempty"`
}
