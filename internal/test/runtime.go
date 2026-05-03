package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// dockerHostEnv returns a DOCKER_HOST=... entry to append to a child process's
// environment, or "" when no override is needed. crossplane render uses the
// Docker SDK; Podman exposes a Docker-compatible socket but the SDK's default
// path is Docker-only, so we point at the Podman socket when that's the only
// runtime present.
func dockerHostEnv() string {
	if os.Getenv("DOCKER_HOST") != "" {
		return ""
	}
	if dockerSocketAvailable() {
		return ""
	}
	if sock := podmanSocket(); sock != "" {
		return "DOCKER_HOST=unix://" + sock
	}
	return ""
}

func dockerSocketAvailable() bool {
	candidates := []string{"/var/run/docker.sock"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".docker/run/docker.sock"))
	}
	for _, p := range candidates {
		if isSocket(p) {
			return true
		}
	}
	return false
}

func podmanSocket() string {
	if out, err := exec.Command("podman", "machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output(); err == nil {
		if p := strings.TrimSpace(string(out)); p != "" && p != "<no value>" && isSocket(p) {
			return p
		}
	}
	if rt := os.Getenv("XDG_RUNTIME_DIR"); rt != "" {
		p := filepath.Join(rt, "podman", "podman.sock")
		if isSocket(p) {
			return p
		}
	}
	if isSocket("/run/podman/podman.sock") {
		return "/run/podman/podman.sock"
	}
	return ""
}

func isSocket(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSocket != 0
}
