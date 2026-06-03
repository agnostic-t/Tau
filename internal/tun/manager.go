package tun

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

type Manager struct {
	mu          sync.Mutex
	tunIF       string
	primalIF    string
	mainGateway string
	savedRoutes []Route
	isEnabled   bool
	serverIP    string
}

type Route struct {
	Raw string
}

func NewManager(tunIF, primalIF, mainGateway, serverIP string) (*Manager, error) {
	if !isAlphanum(tunIF) || !isAlphanum(primalIF) || !isValidIPv4(mainGateway) || !isValidIPv4(serverIP) {
		return nil, errors.New("interfaces/main gateway/server have incorrect values")
	}

	if os.Geteuid() != 0 {
		return nil, errors.New("root privileges required")
	}

	return &Manager{
		tunIF:       tunIF,
		primalIF:    primalIF,
		mainGateway: mainGateway,
		serverIP:    serverIP,
	}, nil
}

func (m *Manager) Enable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isEnabled {
		return nil
	}

	routes, err := getRoutes(m.primalIF)
	if err != nil {
		return fmt.Errorf("failed to save routes: %w", err)
	}
	m.savedRoutes = routes

	steps := [][]string{
		{"ip", "tuntap", "add", "mode", "tun", "dev", m.tunIF},
		{"ip", "addr", "add", "198.18.0.1/15", "dev", m.tunIF},
		{"ip", "link", "set", "dev", m.tunIF, "up"},
		{"ip", "route", "add", m.serverIP, "via", m.mainGateway, "dev", m.primalIF},
		{"ip", "route", "del", "default"},
		{"ip", "route", "add", "default", "via", "198.18.0.1", "dev", m.tunIF, "metric", "1"},
		{"ip", "route", "add", "default", "via", m.mainGateway, "dev", m.primalIF, "metric", "10"},
		{"sysctl", "-w", "net.ipv4.conf.all.rp_filter=0"},
		{"sysctl", "-w", "net.ipv4.conf." + m.primalIF + ".rp_filter=0"},
	}

	for i, args := range steps {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			m.teardown()
			return fmt.Errorf("step %d (%v) failed: %w", i, args, err)
		}
	}

	exec.Command("ip", "-6", "addr", "add", "fd00::1/64", "dev", m.tunIF).Run()
	exec.Command("ip", "-6", "route", "add", "default", "dev", m.tunIF, "metric", "1").Run()

	m.isEnabled = true
	return nil
}

func (m *Manager) Disable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isEnabled {
		return nil
	}

	if err := m.teardown(); err != nil {
		return err
	}

	m.isEnabled = false
	return nil
}

func (m *Manager) teardown() error {
	var errs []error

	exec.Command("ip", "route", "del", "default", "dev", m.tunIF).Run()
	exec.Command("ip", "route", "del", "default", "via", m.mainGateway, "dev", m.primalIF, "metric", "10").Run()

	if err := restoreRoutes(m.savedRoutes); err != nil {
		errs = append(errs, fmt.Errorf("failed to restore routes: %w", err))
	}

	exec.Command("ip", "addr", "del", "198.18.0.1/15", "dev", m.tunIF).Run()

	err := exec.Command("ip", "link", "delete", m.tunIF).Run()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to delete interface %s: %w", m.tunIF, err))
	}

	exec.Command("sysctl", "-w", "net.ipv4.conf.all.rp_filter=1").Run()
	exec.Command("sysctl", "-w", "net.ipv4.conf."+m.primalIF+".rp_filter=1").Run()

	exec.Command("ip", "tuntap", "del", "mode", "tun", "dev", m.tunIF).Run()
	exec.Command("ip", "route", "del", m.serverIP, "via", m.mainGateway, "dev", m.primalIF).Run()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// ---

func getRoutes(dev string) ([]Route, error) {
	out, err := exec.Command("ip", "route", "show", "dev", dev).Output()
	if err != nil {
		return nil, err
	}
	var routes []Route
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			routes = append(routes, Route{Raw: line})
		}
	}
	return routes, nil
}

func restoreRoutes(routes []Route) error {
	for _, r := range routes {
		args := strings.Fields(r.Raw)
		if len(args) > 0 {
			exec.Command(args[0], args[1:]...).Run()
		}
	}
	return nil
}

func isAlphanum(word string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9_-]*$`).MatchString(word)
}

func isValidIPv4(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil
}
