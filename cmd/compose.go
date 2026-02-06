package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/raviatluri/dctl/pkg/compose"
	"github.com/raviatluri/dctl/pkg/runner"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// composeCommands returns the compose command group.
func composeCommands() []*cli.Command {
	composeGlobalFlags := []cli.Flag{
		&cli.StringSliceFlag{Name: "file", Aliases: []string{"f"}, Usage: "Compose configuration files"},
		&cli.StringFlag{Name: "project-name", Aliases: []string{"p"}, Usage: "Project name"},
		&cli.StringFlag{Name: "project-directory", Usage: "Specify an alternate working directory"},
		&cli.StringSliceFlag{Name: "profile", Usage: "Specify a profile to enable"},
		&cli.StringFlag{Name: "env-file", Usage: "Specify an alternate environment file"},
	}
	_ = composeGlobalFlags

	return []*cli.Command{
		{
			Name:  "compose",
			Usage: "Docker Compose compatible commands",
			Flags: composeGlobalFlags,
			Commands: []*cli.Command{
				{
					Name:  "up",
					Usage: "Create and start containers",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "detach", Aliases: []string{"d"}, Usage: "Detached mode: run containers in the background"},
						&cli.BoolFlag{Name: "build", Usage: "Build images before starting containers"},
						&cli.BoolFlag{Name: "force-recreate", Usage: "Recreate containers even if unchanged"},
						&cli.BoolFlag{Name: "remove-orphans", Usage: "Remove containers for undefined services"},
						&cli.IntFlag{Name: "timeout", Aliases: []string{"t"}, Usage: "Shutdown timeout in seconds", Value: 10},
						&cli.BoolFlag{Name: "wait", Usage: "Wait for services to be running/healthy"},
					},
					Action: composeUpAction,
				},
				{
					Name:  "down",
					Usage: "Stop and remove containers, networks",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "volumes", Aliases: []string{"v"}, Usage: "Remove named volumes"},
						&cli.BoolFlag{Name: "remove-orphans", Usage: "Remove containers for undefined services"},
						&cli.IntFlag{Name: "timeout", Aliases: []string{"t"}, Usage: "Shutdown timeout in seconds", Value: 10},
					},
					Action: composeDownAction,
				},
				{
					Name:  "ps",
					Usage: "List containers",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "Only display container IDs"},
						&cli.StringFlag{Name: "format", Usage: "Output format (table|json)"},
					},
					Action: composePsAction,
				},
				{
					Name:      "logs",
					Usage:     "View output from containers",
					ArgsUsage: "[SERVICE...]",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "follow", Aliases: []string{"f"}, Usage: "Follow log output"},
						&cli.StringFlag{Name: "tail", Aliases: []string{"n"}, Usage: "Number of lines from end", Value: "all"},
						&cli.BoolFlag{Name: "timestamps", Aliases: []string{"t"}, Usage: "Show timestamps"},
					},
					Action: composeLogsAction,
				},
				{
					Name:      "exec",
					Usage:     "Execute a command in a running service container",
					ArgsUsage: "SERVICE COMMAND [ARG...]",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "detach", Aliases: []string{"d"}, Usage: "Run in background"},
						&cli.StringSliceFlag{Name: "env", Aliases: []string{"e"}, Usage: "Set environment variables"},
						&cli.BoolFlag{Name: "no-TTY", Aliases: []string{"T"}, Usage: "Disable pseudo-TTY allocation"},
						&cli.StringFlag{Name: "user", Aliases: []string{"u"}, Usage: "Run as this user"},
						&cli.StringFlag{Name: "workdir", Aliases: []string{"w"}, Usage: "Working directory"},
					},
					Action: composeExecAction,
				},
				{
					Name:      "run",
					Usage:     "Run a one-off command on a service",
					ArgsUsage: "SERVICE [COMMAND] [ARG...]",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "detach", Aliases: []string{"d"}, Usage: "Run in background"},
						&cli.BoolFlag{Name: "rm", Usage: "Remove container when it exits"},
						&cli.StringSliceFlag{Name: "env", Aliases: []string{"e"}, Usage: "Set environment variables"},
						&cli.StringSliceFlag{Name: "publish", Aliases: []string{"p"}, Usage: "Publish port(s)"},
						&cli.StringFlag{Name: "user", Aliases: []string{"u"}, Usage: "Run as this user"},
						&cli.StringSliceFlag{Name: "volume", Aliases: []string{"v"}, Usage: "Bind mount a volume"},
						&cli.StringFlag{Name: "workdir", Aliases: []string{"w"}, Usage: "Working directory"},
						&cli.BoolFlag{Name: "no-deps", Usage: "Don't start linked services"},
						&cli.StringFlag{Name: "name", Usage: "Assign a name to the container"},
						&cli.StringFlag{Name: "entrypoint", Usage: "Override the entrypoint"},
					},
					Action: composeRunAction,
				},
				{
					Name:  "build",
					Usage: "Build or rebuild services",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "no-cache", Usage: "Do not use cache"},
						&cli.BoolFlag{Name: "pull", Usage: "Always pull a newer version of the image"},
						&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "Don't print anything to STDOUT"},
						&cli.StringSliceFlag{Name: "build-arg", Usage: "Set build-time variables"},
					},
					Action: composeBuildAction,
				},
				{
					Name:   "pull",
					Usage:  "Pull service images",
					Action: composePullAction,
				},
				{
					Name:      "stop",
					Usage:     "Stop services",
					ArgsUsage: "[SERVICE...]",
					Flags: []cli.Flag{
						&cli.IntFlag{Name: "timeout", Aliases: []string{"t"}, Usage: "Shutdown timeout in seconds", Value: 10},
					},
					Action: composeStopAction,
				},
				{
					Name:      "restart",
					Usage:     "Restart service containers",
					ArgsUsage: "[SERVICE...]",
					Flags: []cli.Flag{
						&cli.IntFlag{Name: "timeout", Aliases: []string{"t"}, Usage: "Shutdown timeout in seconds", Value: 10},
					},
					Action: composeRestartAction,
				},
				{
					Name:  "config",
					Usage: "Parse, resolve and render compose file",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "Only validate, don't print"},
					},
					Action: composeConfigAction,
				},
				{
					Name:      "rm",
					Usage:     "Remove stopped service containers",
					ArgsUsage: "[SERVICE...]",
					Flags: []cli.Flag{
						&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "Don't ask to confirm removal"},
						&cli.BoolFlag{Name: "stop", Aliases: []string{"s"}, Usage: "Stop containers before removing"},
						&cli.BoolFlag{Name: "volumes", Aliases: []string{"v"}, Usage: "Remove anonymous volumes"},
					},
					Action: composeRmAction,
				},
				{
					Name:      "kill",
					Usage:     "Force stop service containers",
					ArgsUsage: "[SERVICE...]",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "signal", Aliases: []string{"s"}, Usage: "Signal to send", Value: "SIGKILL"},
					},
					Action: composeKillAction,
				},
			},
		},
	}
}

// --- Compose helpers ---

// composeContext extracts project directory, compose file, and project name
// from the global compose flags.
type composeContext struct {
	projectDir  string
	composeFile *compose.ComposeFile
	projectName string
}

// resolveComposeContext loads compose files and resolves the project name.
func resolveComposeContext(cmd *cli.Command) (*composeContext, error) {
	projectDir := cmd.String("project-directory")
	if projectDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		projectDir = wd
	}

	files := cmd.StringSlice("file")

	cf, err := compose.Load(files, projectDir)
	if err != nil {
		return nil, err
	}

	projectName := compose.ResolveProjectName(cmd.String("project-name"), cf, projectDir)

	return &composeContext{
		projectDir:  projectDir,
		composeFile: cf,
		projectName: projectName,
	}, nil
}

// containerName returns the container name for a service in a project.
func containerName(project, service string) string {
	return project + "_" + service
}

// buildRunArgs constructs container run arguments from a compose.Service definition.
func buildRunArgs(svc compose.Service, project, svcName string) []string {
	name := containerName(project, svcName)
	args := []string{"run", "--detach", "--name", name}

	// ports
	for _, p := range svc.Ports {
		args = append(args, "--publish", p)
	}

	// volumes
	for _, v := range svc.Volumes {
		args = append(args, "--volume", v)
	}

	// environment
	if env, ok := svc.Environment.(map[string]string); ok {
		for k, v := range env {
			args = append(args, "--env", k+"="+v)
		}
	}

	// working_dir
	if svc.WorkingDir != "" {
		args = append(args, "--workdir", svc.WorkingDir)
	}

	// user
	if svc.User != "" {
		args = append(args, "--user", svc.User)
	}

	// tty
	if svc.Tty {
		args = append(args, "--tty")
	}

	// stdin_open
	if svc.StdinOpen {
		args = append(args, "--interactive")
	}

	// read_only
	if svc.ReadOnly {
		args = append(args, "--read-only")
	}

	// cpus
	if svc.CPUs != nil {
		args = append(args, "--cpus", fmt.Sprintf("%v", svc.CPUs))
	}

	// mem_limit
	if svc.MemLimit != "" {
		args = append(args, "--memory", svc.MemLimit)
	}

	// dns
	if dns, ok := svc.DNS.([]string); ok {
		for _, d := range dns {
			args = append(args, "--dns", d)
		}
	}

	// labels
	for k, v := range svc.Labels {
		args = append(args, "--label", k+"="+v)
	}

	// tmpfs
	if tmpfs, ok := svc.Tmpfs.([]string); ok {
		for _, t := range tmpfs {
			args = append(args, "--tmpfs", t)
		}
	}

	// entrypoint
	if ep, ok := svc.Entrypoint.([]string); ok && len(ep) > 0 {
		args = append(args, "--entrypoint", ep[0])
	}

	// platform
	if svc.Platform != "" {
		args = append(args, "--platform", svc.Platform)
	}

	// network (first network key)
	if nets, ok := svc.Networks.(map[string]interface{}); ok {
		for netName := range nets {
			args = append(args, "--network", netName)
			break
		}
	}

	// image (required positional arg)
	args = append(args, svc.Image)

	// command
	if cmdSlice, ok := svc.Command.([]string); ok {
		args = append(args, cmdSlice...)
	}

	return args
}

// filterServices returns the list of services to operate on.
// If args are given, uses those; otherwise returns all services from state.
func filterServices(state *compose.ProjectState, args []string) []string {
	if len(args) > 0 {
		return args
	}
	services := make([]string, 0, len(state.Containers))
	for svc := range state.Containers {
		services = append(services, svc)
	}
	return services
}

// --- Compose actions ---

func composeUpAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	cf := cc.composeFile
	project := cc.projectName

	// Create networks
	var createdNetworks []string
	for name, net := range cf.Networks {
		if net.External {
			continue
		}
		netName := name
		if net.Name != "" {
			netName = net.Name
		}
		fmt.Fprintf(os.Stderr, "Creating network %s\n", netName)
		createArgs := []string{"network", "create", netName}
		if err := runner.Run(createArgs...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create network %s: %v\n", netName, err)
		} else {
			createdNetworks = append(createdNetworks, netName)
		}
	}

	// Create volumes
	var createdVolumes []string
	for name, vol := range cf.Volumes {
		if vol.External {
			continue
		}
		volName := name
		if vol.Name != "" {
			volName = vol.Name
		}
		fmt.Fprintf(os.Stderr, "Creating volume %s\n", volName)
		createArgs := []string{"volume", "create", volName}
		if err := runner.Run(createArgs...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create volume %s: %v\n", volName, err)
		} else {
			createdVolumes = append(createdVolumes, volName)
		}
	}

	// Build images if --build flag is set
	if cmd.Bool("build") {
		for svcName, svc := range cf.Services {
			bc, ok := svc.Build.(*compose.BuildConfig)
			if !ok || bc == nil {
				continue
			}
			fmt.Fprintf(os.Stderr, "Building %s\n", svcName)
			buildArgs := composeBuildCLIArgs(bc, svc.Image, cc.projectDir)
			if err := runner.Run(buildArgs...); err != nil {
				return fmt.Errorf("building service %s: %w", svcName, err)
			}
		}
	}

	// Resolve startup order
	order, err := compose.ResolveOrder(cf.Services)
	if err != nil {
		return err
	}

	// Start containers in order
	containers := make(map[string]string)
	var startedServices []string
	for _, svcName := range order {
		svc := cf.Services[svcName]
		if svc.Image == "" {
			if bc, ok := svc.Build.(*compose.BuildConfig); ok && bc != nil {
				svc.Image = project + "-" + svcName
			} else {
				return fmt.Errorf("service %s has no image and no build config", svcName)
			}
		}

		cName := containerName(project, svcName)
		fmt.Fprintf(os.Stderr, "Starting %s\n", cName)

		runArgs := buildRunArgs(svc, project, svcName)
		if err := runner.Run(runArgs...); err != nil {
			// Rollback: stop already-started services
			fmt.Fprintf(os.Stderr, "Failed to start %s, stopping started services\n", cName)
			for i := len(startedServices) - 1; i >= 0; i-- {
				stopName := containerName(project, startedServices[i])
				_ = runner.Run("stop", stopName)
			}
			return fmt.Errorf("starting service %s: %w", svcName, err)
		}
		startedServices = append(startedServices, svcName)
		containers[svcName] = cName
	}

	// Determine compose file path for state
	composeFilePath := ""
	files := cmd.StringSlice("file")
	if len(files) > 0 {
		composeFilePath = files[0]
	}

	// Save project state
	state := &compose.ProjectState{
		Name:        project,
		ComposeFile: composeFilePath,
		ProjectDir:  cc.projectDir,
		Containers:  containers,
		Networks:    createdNetworks,
		Volumes:     createdVolumes,
	}
	if err := compose.SaveProject(state); err != nil {
		return fmt.Errorf("saving project state: %w", err)
	}

	return nil
}

func composeDownAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	state, err := compose.LoadProject(cc.projectName)
	if err != nil {
		return err
	}

	// Stop and remove all containers
	for svcName, cName := range state.Containers {
		fmt.Fprintf(os.Stderr, "Stopping %s\n", cName)
		if err := runner.Run("stop", cName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to stop %s: %v\n", svcName, err)
		}
		fmt.Fprintf(os.Stderr, "Removing %s\n", cName)
		if err := runner.Run("delete", cName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", svcName, err)
		}
	}

	// Remove volumes if --volumes flag
	if cmd.Bool("volumes") {
		for _, vol := range state.Volumes {
			fmt.Fprintf(os.Stderr, "Removing volume %s\n", vol)
			if err := runner.Run("volume", "delete", vol); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove volume %s: %v\n", vol, err)
			}
		}
	}

	// Remove networks
	for _, net := range state.Networks {
		fmt.Fprintf(os.Stderr, "Removing network %s\n", net)
		if err := runner.Run("network", "delete", net); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove network %s: %v\n", net, err)
		}
	}

	// Delete project state
	if err := compose.DeleteProject(cc.projectName); err != nil {
		return fmt.Errorf("deleting project state: %w", err)
	}

	return nil
}

func composePsAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	state, err := compose.LoadProject(cc.projectName)
	if err != nil {
		return err
	}

	// Get all containers in JSON format
	out, err := runner.Output("list", "--format", "json")
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	if out == "" {
		return nil
	}

	// Build set of our container names
	projectContainers := make(map[string]bool)
	for _, cName := range state.Containers {
		projectContainers[cName] = true
	}

	// Parse and filter JSON output
	// The output may be a JSON array or newline-delimited JSON objects
	var allContainers []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &allContainers); err != nil {
		// Try newline-delimited
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var c map[string]interface{}
			if err := json.Unmarshal([]byte(line), &c); err != nil {
				continue
			}
			allContainers = append(allContainers, c)
		}
	}

	// Filter to project containers and print
	for _, c := range allContainers {
		name, _ := c["Name"].(string)
		if name == "" {
			name, _ = c["name"].(string)
		}
		if projectContainers[name] {
			data, _ := json.Marshal(c)
			fmt.Println(string(data))
		}
	}

	return nil
}

func composeLogsAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	state, err := compose.LoadProject(cc.projectName)
	if err != nil {
		return err
	}

	services := filterServices(state, cmd.Args().Slice())

	for _, svcName := range services {
		cName, ok := state.Containers[svcName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: no container found for service %s\n", svcName)
			continue
		}

		args := []string{"logs"}
		if cmd.Bool("follow") {
			args = append(args, "--follow")
		}
		if n := cmd.String("tail"); n != "" && n != "all" {
			args = append(args, "-n", n)
		}
		args = append(args, cName)

		if err := runner.Run(args...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get logs for %s: %v\n", svcName, err)
		}
	}

	return nil
}

func composeExecAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 2 {
		return fmt.Errorf("requires at least 2 arguments: SERVICE COMMAND [ARG...]")
	}

	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	state, err := compose.LoadProject(cc.projectName)
	if err != nil {
		return err
	}

	svcName := cmd.Args().First()
	execArgs := cmd.Args().Tail()

	cName, ok := state.Containers[svcName]
	if !ok {
		return fmt.Errorf("no container found for service %s", svcName)
	}

	args := []string{"exec"}
	if cmd.Bool("detach") {
		args = append(args, "--detach")
	}
	if !cmd.Bool("no-TTY") {
		args = append(args, "--tty")
	}
	if u := cmd.String("user"); u != "" {
		args = append(args, "--user", u)
	}
	if w := cmd.String("workdir"); w != "" {
		args = append(args, "--workdir", w)
	}
	for _, e := range cmd.StringSlice("env") {
		args = append(args, "--env", e)
	}
	args = append(args, cName)
	args = append(args, execArgs...)

	return runner.Run(args...)
}

func composeRunAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("requires at least 1 argument: SERVICE [COMMAND] [ARG...]")
	}

	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	cf := cc.composeFile
	project := cc.projectName
	svcName := cmd.Args().First()
	cmdArgs := cmd.Args().Tail()

	svc, ok := cf.Services[svcName]
	if !ok {
		return fmt.Errorf("no such service: %s", svcName)
	}

	if svc.Image == "" {
		if bc, ok := svc.Build.(*compose.BuildConfig); ok && bc != nil {
			svc.Image = project + "-" + svcName
		} else {
			return fmt.Errorf("service %s has no image and no build config", svcName)
		}
	}

	// Override command if provided
	if len(cmdArgs) > 0 {
		svc.Command = cmdArgs
	}

	// Build run args from service config
	name := containerName(project, svcName) + "_run"
	if n := cmd.String("name"); n != "" {
		name = n
	}
	args := []string{"run"}
	if cmd.Bool("detach") {
		args = append(args, "--detach")
	}
	if cmd.Bool("rm") {
		args = append(args, "--rm")
	}
	args = append(args, "--name", name)

	// Ports from service, overridden by flag
	ports := svc.Ports
	if flagPorts := cmd.StringSlice("publish"); len(flagPorts) > 0 {
		ports = flagPorts
	}
	for _, p := range ports {
		args = append(args, "--publish", p)
	}

	// Volumes from service, plus flag overrides
	for _, v := range svc.Volumes {
		args = append(args, "--volume", v)
	}
	for _, v := range cmd.StringSlice("volume") {
		args = append(args, "--volume", v)
	}

	// Environment from service, plus flag overrides
	if env, ok := svc.Environment.(map[string]string); ok {
		for k, v := range env {
			args = append(args, "--env", k+"="+v)
		}
	}
	for _, e := range cmd.StringSlice("env") {
		args = append(args, "--env", e)
	}

	// User
	user := svc.User
	if u := cmd.String("user"); u != "" {
		user = u
	}
	if user != "" {
		args = append(args, "--user", user)
	}

	// Workdir
	workdir := svc.WorkingDir
	if w := cmd.String("workdir"); w != "" {
		workdir = w
	}
	if workdir != "" {
		args = append(args, "--workdir", workdir)
	}

	// Entrypoint
	if ep := cmd.String("entrypoint"); ep != "" {
		args = append(args, "--entrypoint", ep)
	} else if ep, ok := svc.Entrypoint.([]string); ok && len(ep) > 0 {
		args = append(args, "--entrypoint", ep[0])
	}

	if svc.Tty {
		args = append(args, "--tty")
	}
	if svc.StdinOpen {
		args = append(args, "--interactive")
	}

	// Network
	if nets, ok := svc.Networks.(map[string]interface{}); ok {
		for netName := range nets {
			args = append(args, "--network", netName)
			break
		}
	}

	// Platform
	if svc.Platform != "" {
		args = append(args, "--platform", svc.Platform)
	}

	args = append(args, svc.Image)

	// Command args
	if cmdSlice, ok := svc.Command.([]string); ok {
		args = append(args, cmdSlice...)
	}

	return runner.Run(args...)
}

func composeBuildAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	cf := cc.composeFile
	project := cc.projectName

	services := cmd.Args().Slice()
	if len(services) == 0 {
		for name := range cf.Services {
			services = append(services, name)
		}
	}

	for _, svcName := range services {
		svc, ok := cf.Services[svcName]
		if !ok {
			return fmt.Errorf("no such service: %s", svcName)
		}

		bc, ok := svc.Build.(*compose.BuildConfig)
		if !ok || bc == nil {
			fmt.Fprintf(os.Stderr, "Skipping %s: no build config\n", svcName)
			continue
		}

		tag := svc.Image
		if tag == "" {
			tag = project + "-" + svcName
		}

		fmt.Fprintf(os.Stderr, "Building %s\n", svcName)
		buildArgs := composeBuildCLIArgs(bc, tag, cc.projectDir)

		// Add CLI flag overrides
		if cmd.Bool("no-cache") {
			buildArgs = append(buildArgs, "--no-cache")
		}
		for _, arg := range cmd.StringSlice("build-arg") {
			buildArgs = append(buildArgs, "--build-arg", arg)
		}

		if err := runner.Run(buildArgs...); err != nil {
			return fmt.Errorf("building service %s: %w", svcName, err)
		}
	}

	return nil
}

// composeBuildCLIArgs builds container build CLI arguments from a BuildConfig.
func composeBuildCLIArgs(bc *compose.BuildConfig, tag, projectDir string) []string {
	args := []string{"build"}

	if tag != "" {
		args = append(args, "--tag", tag)
	}
	if bc.Dockerfile != "" {
		args = append(args, "--file", bc.Dockerfile)
	}
	if bc.Target != "" {
		args = append(args, "--target", bc.Target)
	}
	for k, v := range bc.Args {
		args = append(args, "--build-arg", k+"="+v)
	}
	for k, v := range bc.Labels {
		args = append(args, "--label", k+"="+v)
	}

	buildContext := bc.Context
	if buildContext == "" {
		buildContext = "."
	}
	if !filepath.IsAbs(buildContext) {
		buildContext = filepath.Join(projectDir, buildContext)
	}
	args = append(args, buildContext)

	return args
}

func composePullAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	cf := cc.composeFile

	services := cmd.Args().Slice()
	if len(services) == 0 {
		for name := range cf.Services {
			services = append(services, name)
		}
	}

	for _, svcName := range services {
		svc, ok := cf.Services[svcName]
		if !ok {
			return fmt.Errorf("no such service: %s", svcName)
		}
		if svc.Image == "" {
			fmt.Fprintf(os.Stderr, "Skipping %s: no image defined\n", svcName)
			continue
		}
		fmt.Fprintf(os.Stderr, "Pulling %s\n", svc.Image)
		if err := runner.Run("image", "pull", svc.Image); err != nil {
			return fmt.Errorf("pulling image for %s: %w", svcName, err)
		}
	}

	return nil
}

func composeStopAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	state, err := compose.LoadProject(cc.projectName)
	if err != nil {
		return err
	}

	services := filterServices(state, cmd.Args().Slice())

	for _, svcName := range services {
		cName, ok := state.Containers[svcName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: no container found for service %s\n", svcName)
			continue
		}
		fmt.Fprintf(os.Stderr, "Stopping %s\n", cName)
		if err := runner.Run("stop", cName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to stop %s: %v\n", svcName, err)
		}
	}

	return nil
}

func composeRestartAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	state, err := compose.LoadProject(cc.projectName)
	if err != nil {
		return err
	}

	services := filterServices(state, cmd.Args().Slice())

	// Stop services
	for _, svcName := range services {
		cName, ok := state.Containers[svcName]
		if !ok {
			continue
		}
		fmt.Fprintf(os.Stderr, "Stopping %s\n", cName)
		if err := runner.Run("stop", cName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to stop %s: %v\n", svcName, err)
		}
	}

	// Start services
	for _, svcName := range services {
		cName, ok := state.Containers[svcName]
		if !ok {
			continue
		}
		fmt.Fprintf(os.Stderr, "Starting %s\n", cName)
		if err := runner.Run("start", cName); err != nil {
			return fmt.Errorf("starting %s: %w", svcName, err)
		}
	}

	return nil
}

func composeConfigAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	if cmd.Bool("quiet") {
		// Just validate, don't print
		return nil
	}

	out, err := yaml.Marshal(cc.composeFile)
	if err != nil {
		return fmt.Errorf("marshaling compose file: %w", err)
	}
	fmt.Print(string(out))
	return nil
}

func composeRmAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	state, err := compose.LoadProject(cc.projectName)
	if err != nil {
		return err
	}

	services := filterServices(state, cmd.Args().Slice())

	// Optionally stop first
	if cmd.Bool("stop") {
		for _, svcName := range services {
			cName, ok := state.Containers[svcName]
			if !ok {
				continue
			}
			fmt.Fprintf(os.Stderr, "Stopping %s\n", cName)
			_ = runner.Run("stop", cName)
		}
	}

	for _, svcName := range services {
		cName, ok := state.Containers[svcName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: no container found for service %s\n", svcName)
			continue
		}
		fmt.Fprintf(os.Stderr, "Removing %s\n", cName)
		deleteArgs := []string{"delete"}
		if cmd.Bool("force") {
			deleteArgs = append(deleteArgs, "--force")
		}
		deleteArgs = append(deleteArgs, cName)
		if err := runner.Run(deleteArgs...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", svcName, err)
		}
	}

	return nil
}

func composeKillAction(ctx context.Context, cmd *cli.Command) error {
	cc, err := resolveComposeContext(cmd)
	if err != nil {
		return err
	}

	state, err := compose.LoadProject(cc.projectName)
	if err != nil {
		return err
	}

	services := filterServices(state, cmd.Args().Slice())
	signal := cmd.String("signal")

	for _, svcName := range services {
		cName, ok := state.Containers[svcName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: no container found for service %s\n", svcName)
			continue
		}
		fmt.Fprintf(os.Stderr, "Killing %s\n", cName)
		killArgs := []string{"kill"}
		if signal != "" && signal != "SIGKILL" {
			killArgs = append(killArgs, "--signal", signal)
		}
		killArgs = append(killArgs, cName)
		if err := runner.Run(killArgs...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to kill %s: %v\n", svcName, err)
		}
	}

	return nil
}
