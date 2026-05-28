package router

import (
	"fmt"
	"net/http"
	"strings"
)

type Route struct {
	Method  string
	Pattern string
	Module  string
}

type Router interface {
	HandleFunc(pattern string, handler func(w http.ResponseWriter, r *http.Request))
	Module(name string) Router
}

type TrackedMux struct {
	*http.ServeMux
	routes        []Route
	currentModule string
}

func NewTrackedMux() *TrackedMux {
	return &TrackedMux{ServeMux: http.NewServeMux()}
}

func (m *TrackedMux) Module(name string) Router {
	m.currentModule = name
	return m
}

func (m *TrackedMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	method, path, _ := strings.Cut(pattern, " ")
	if path == "" {
		path = method
		method = "ANY"
	}
	m.routes = append(m.routes, Route{
		Method:  method,
		Pattern: path,
		Module:  m.currentModule,
	})
	m.ServeMux.HandleFunc(pattern, handler)
}

func (m *TrackedMux) PrintRoutes(startupMs int64) {
	methodColor := map[string]string{
		"GET":    "\033[32m",
		"POST":   "\033[34m",
		"PUT":    "\033[33m",
		"PATCH":  "\033[35m",
		"DELETE": "\033[31m",
		"ANY":    "\033[36m",
	}
	reset := "\033[0m"
	bold := "\033[1m"
	gray := "\033[90m"
	yellow := "\033[33m"

	// Agrupa por módulo manteniendo orden de inserción
	type moduleGroup struct {
		name   string
		routes []Route
	}
	seen := map[string]int{}
	groups := []moduleGroup{}

	for _, r := range m.routes {
		mod := r.Module
		if mod == "" {
			mod = "core"
		}
		if idx, ok := seen[mod]; ok {
			groups[idx].routes = append(groups[idx].routes, r)
		} else {
			seen[mod] = len(groups)
			groups = append(groups, moduleGroup{name: mod, routes: []Route{r}})
		}
	}

	width := 42
	line := strings.Repeat("─", width)

	fmt.Println()
	fmt.Printf("  %s⬡  Honux Core%s\n", bold, reset)
	fmt.Printf("  %s%s%s\n", gray, line, reset)

	for _, g := range groups {
		fmt.Printf("\n  %s[%s]%s\n", yellow+bold, strings.ToUpper(g.name), reset)
		for _, r := range g.routes {
			color := methodColor[r.Method]
			if color == "" {
				color = reset
			}
			fmt.Printf("    %s%-7s%s %s\n", color, r.Method, reset, r.Pattern)
		}
	}

	fmt.Printf("\n  %s%s%s\n", gray, line, reset)
	fmt.Printf("  %s✓ Server ready in %dms%s\n\n", bold, startupMs, reset)
}
