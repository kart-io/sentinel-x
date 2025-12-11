package bootstrap

import (
	"fmt"
	"strings"
)

// ResolveDependencies performs topological sort on initializers based on their dependencies.
// Returns initializers in the order they should be executed.
// Returns error if circular dependency is detected or if a dependency is not found.
func ResolveDependencies(initializers []Initializer) ([]Initializer, error) {
	if len(initializers) == 0 {
		return nil, nil
	}

	// 构建名称到初始化器的映射
	nameToInit := make(map[string]Initializer)
	for _, init := range initializers {
		if _, exists := nameToInit[init.Name()]; exists {
			return nil, fmt.Errorf("duplicate initializer name: %s", init.Name())
		}
		nameToInit[init.Name()] = init
	}

	// 构建依赖图
	graph := make(map[string][]string)
	for _, init := range initializers {
		graph[init.Name()] = init.Dependencies()
	}

	// 验证所有依赖都存在
	for name, deps := range graph {
		for _, dep := range deps {
			if _, exists := nameToInit[dep]; !exists {
				return nil, fmt.Errorf("initializer %q depends on %q which is not registered", name, dep)
			}
		}
	}

	// 检测循环依赖
	if err := validateNoCycles(graph); err != nil {
		return nil, err
	}

	// 拓扑排序
	return topologicalSort(initializers, graph)
}

// validateNoCycles checks for circular dependencies in the graph.
// Uses DFS with three-color marking: white (unvisited), gray (in progress), black (done).
func validateNoCycles(graph map[string][]string) error {
	const (
		white = 0 // 未访问
		gray  = 1 // 正在访问（在当前 DFS 路径中）
		black = 2 // 已完成
	)

	color := make(map[string]int)
	parent := make(map[string]string) // 用于构建循环路径

	var dfs func(node string) error
	dfs = func(node string) error {
		color[node] = gray

		for _, dep := range graph[node] {
			if color[dep] == gray {
				// 发现循环，构建循环路径
				cycle := buildCyclePath(node, dep, parent)
				return fmt.Errorf("circular dependency detected: %s", cycle)
			}
			if color[dep] == white {
				parent[dep] = node
				if err := dfs(dep); err != nil {
					return err
				}
			}
		}

		color[node] = black
		return nil
	}

	for name := range graph {
		if color[name] == white {
			if err := dfs(name); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildCyclePath builds a string representation of the cycle path.
func buildCyclePath(from, to string, parent map[string]string) string {
	path := []string{to}
	current := from
	for current != to {
		path = append(path, current)
		current = parent[current]
		if current == "" {
			break
		}
	}
	path = append(path, to)

	// 反转路径
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return strings.Join(path, " -> ")
}

// topologicalSort performs Kahn's algorithm to sort initializers by dependencies.
func topologicalSort(initializers []Initializer, graph map[string][]string) ([]Initializer, error) {
	// 计算入度
	inDegree := make(map[string]int)
	for _, init := range initializers {
		inDegree[init.Name()] = 0
	}

	// 构建反向依赖图（谁依赖我）
	reverseDeps := make(map[string][]string)
	for name, deps := range graph {
		for _, dep := range deps {
			reverseDeps[dep] = append(reverseDeps[dep], name)
			inDegree[name]++
		}
	}

	// 找到所有入度为 0 的节点
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// 名称到初始化器的映射
	nameToInit := make(map[string]Initializer)
	for _, init := range initializers {
		nameToInit[init.Name()] = init
	}

	// 拓扑排序
	var result []Initializer
	for len(queue) > 0 {
		// 取出一个入度为 0 的节点
		name := queue[0]
		queue = queue[1:]
		result = append(result, nameToInit[name])

		// 更新依赖我的节点的入度
		for _, dependent := range reverseDeps[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(initializers) {
		return nil, fmt.Errorf("topological sort failed: some initializers could not be sorted")
	}

	return result, nil
}

// ValidateDependencies validates that all dependencies are correctly declared.
// This is a helper function for testing and debugging.
func ValidateDependencies(initializers []Initializer) error {
	_, err := ResolveDependencies(initializers)
	return err
}
