package compose

import (
	"fmt"
	"sort"
)

// ResolveOrder performs a topological sort on services based on depends_on relationships.
// Returns services in startup order (dependencies first). Detects cycles.
func ResolveOrder(services map[string]Service) ([]string, error) {
	// Build adjacency list: service -> list of services it depends on.
	deps := make(map[string][]string)
	for name := range services {
		deps[name] = nil
	}

	for name, svc := range services {
		if svc.DependsOn == nil {
			continue
		}
		switch d := svc.DependsOn.(type) {
		case map[string]DependsOnCondition:
			for dep := range d {
				if _, ok := services[dep]; !ok {
					return nil, fmt.Errorf("service %q depends on undefined service %q", name, dep)
				}
				deps[name] = append(deps[name], dep)
			}
		}
	}

	// Sort dependency lists for deterministic output.
	for name := range deps {
		sort.Strings(deps[name])
	}

	// Kahn's algorithm for topological sort.
	// Compute in-degree (number of dependencies for each service).
	inDegree := make(map[string]int)
	// Reverse adjacency: who depends on me?
	dependents := make(map[string][]string)
	for name := range services {
		inDegree[name] = 0
	}
	for name, depList := range deps {
		inDegree[name] = len(depList)
		for _, dep := range depList {
			dependents[dep] = append(dependents[dep], name)
		}
	}

	// Start with services that have no dependencies.
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	var order []string
	for len(queue) > 0 {
		// Pop front.
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		// Reduce in-degree for dependents.
		children := dependents[current]
		sort.Strings(children)
		for _, child := range children {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	if len(order) != len(services) {
		// Find services involved in the cycle for a better error message.
		var cycled []string
		for name, degree := range inDegree {
			if degree > 0 {
				cycled = append(cycled, name)
			}
		}
		sort.Strings(cycled)
		return nil, fmt.Errorf("dependency cycle detected among services: %v", cycled)
	}

	return order, nil
}
