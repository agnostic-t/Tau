package routing

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/agnostic-t/neutrino-core/local"
)

var _ local.Filter = &RoutingFilter{}

type RoutingFilter struct {
	directRoutes []*regexp.Regexp
	blockRoutes  []*regexp.Regexp
}

func NewRoutingFilter(directPath, blockPath string) (*RoutingFilter, error) {
	f := &RoutingFilter{
		directRoutes: make([]*regexp.Regexp, 0),
		blockRoutes:  make([]*regexp.Regexp, 0),
	}

	var directStrs []string
	var blockStrs []string

	content, err := os.ReadFile(directPath)
	if err == nil {
		directStrs = strings.Split(string(content), "\n")
	} else {
		fmt.Printf("WARNING: direct path failed to open, direct list is empty\n")
		blockStrs = make([]string, 0)
	}

	content, err = os.ReadFile(blockPath)
	if err == nil {
		blockStrs = strings.Split(string(content), "\n")
	} else {
		fmt.Printf("WARNING: block path failed to open, block list is empty\n")
		blockStrs = make([]string, 0)
	}

	for _, pattern := range directStrs {
		if len(pattern) == 0 {
			continue
		}

		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid direct route regexp %q: %w", pattern, err)
		}
		f.directRoutes = append(f.directRoutes, re)
	}

	for _, pattern := range blockStrs {
		if len(pattern) == 0 {
			continue
		}

		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid block route regexp %q: %w", pattern, err)
		}
		f.blockRoutes = append(f.blockRoutes, re)
	}

	fmt.Printf("[routing] Loaded %d direct and %d block rules\n", len(f.directRoutes), len(f.blockRoutes))

	return f, nil
}

func (f *RoutingFilter) Filter(target string) local.RouteAction {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}

	for _, re := range f.directRoutes {
		if re.MatchString(host) {
			fmt.Println("Direct:", host, "target:", target)
			return local.RouteDirect
		}
	}

	for _, re := range f.blockRoutes {
		if re.MatchString(host) {
			fmt.Println("Blocking:", host, "target:", target)
			return local.RouteBlock
		}
	}

	return local.RouteProxy
}

type DummyFilter struct{}

func (f *DummyFilter) Filter(target string) local.RouteAction {
	return local.RouteProxy
}
