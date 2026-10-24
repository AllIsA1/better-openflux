// Copyright (c) 2026 AllIsA1.
// SPDX-License-Identifier: MIT

// The OpenFlux core this plugin keeps in the panel, and the logic that puts it there.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// coreBuild is one reviewed build of the OpenFlux exit (p1neappleXpress/OpenFlux, GPL-3.0-or-later), built by
// this repository's workflow from the upstream commit and published as a release of this repository. Updating
// the core means a new plugin version with a new row here, which is also what the catalog reviews.
type coreBuild struct {
	Version string            // the upstream tag
	Commit  string            // the commit that tag points at
	Release string            // the release of this repository that holds the binaries
	SHA256  map[string]string // by architecture
}

var pinned = coreBuild{
	Version: "v0.13.0",
	Commit:  "4f9c786cdd1d9fcd0c133e2e0fec0c5a12522610",
	Release: "core-v0.13.0-1",
	SHA256: map[string]string{
		"amd64": "133e0c17eb07e5abf8effd0d563de1f38f18cf65c66f0124ba8bc00fb5b0051d",
		"arm64": "beb496161cccf146e0527c0edd30f7d9a2f77c2a1f5376d2c04286f82b96c179",
	},
}

func (c coreBuild) url(arch string) string {
	return "https://github.com/AllIsA1/better-openflux/releases/download/" + c.Release + "/openflux-linux-" + arch
}

const (
	checkEvery = 6 * time.Hour
	retryAfter = 10 * time.Minute
	firstCheck = 20 * time.Second
	callBudget = 6 * time.Minute // the panel downloads the file itself, which can take a while
)

// state is what the plugin remembers in its data directory.
type state struct {
	Version     string    `json:"version"` // the core it installed
	SHA256      string    `json:"sha256"`  // and its hash
	InstalledAt time.Time `json:"installedAt"`
	CheckedAt   time.Time `json:"checkedAt"`
	LastError   string    `json:"lastError,omitempty"`
}

type panelAPI interface {
	Do(ctx context.Context, method, path string, in, out any) error
}

type panelStatus struct {
	BinaryExists bool   `json:"binaryExists"`
	BinaryPath   string `json:"binaryPath"`
	Probe        struct {
		Running bool `json:"running"`
	} `json:"probe"`
}

// updater keeps the panel's OpenFlux core in step with the pinned build.
type updater struct {
	api   panelAPI
	arch  string
	path  string // state file
	build coreBuild
	now   func() time.Time

	mu sync.Mutex
	st state
}

func newUpdater(api panelAPI, arch, statePath string, build coreBuild) *updater {
	u := &updater{api: api, arch: arch, path: statePath, build: build, now: time.Now}
	if raw, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(raw, &u.st)
	}
	return u
}

func (u *updater) save() {
	raw, err := json.MarshalIndent(u.st, "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(u.path), 0o755)
	}
	if err == nil {
		tmp := u.path + ".tmp"
		if err = os.WriteFile(tmp, append(raw, '\n'), 0o600); err == nil {
			err = os.Rename(tmp, u.path)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "better-openflux: cannot save the state: %v\n", err)
	}
}

// snapshot is what the page shows.
type snapshot struct {
	Want      string    `json:"want"`      // the version the plugin keeps
	Installed string    `json:"installed"` // what it installed last ("" none)
	Current   bool      `json:"current"`   // the panel has that core
	Present   bool      `json:"present"`   // the panel has a core at all
	Running   bool      `json:"running"`
	Arch      string    `json:"arch"`
	CheckedAt time.Time `json:"checkedAt"`
	LastError string    `json:"lastError"`
}

func (u *updater) snapshot(ctx context.Context) snapshot {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := snapshot{Want: u.build.Version, Installed: u.st.Version, Arch: u.arch, CheckedAt: u.st.CheckedAt, LastError: u.st.LastError}
	var ps panelStatus
	if err := u.api.Do(ctx, http.MethodGet, "/tunnel/openflux/status", nil, &ps); err == nil {
		s.Present, s.Running = ps.BinaryExists, ps.Probe.Running
		s.Current = ps.BinaryExists && u.st.SHA256 == u.build.SHA256[u.arch]
	}
	return s
}

// ensure puts the pinned core in the panel when it is not there. Without force it leaves alone a core it has
// already installed (a file the operator put there since is theirs); with force it installs again.
func (u *updater) ensure(ctx context.Context, force bool) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	want := u.build.SHA256[u.arch]
	if want == "" {
		return fmt.Errorf("no core is pinned for %s", u.arch)
	}
	err := u.install(ctx, want, force)
	u.st.CheckedAt = u.now().UTC()
	if err != nil {
		u.st.LastError = err.Error()
	} else {
		u.st.LastError = ""
	}
	u.save()
	return err
}

func (u *updater) install(ctx context.Context, want string, force bool) error {
	var ps panelStatus
	if err := u.api.Do(ctx, http.MethodGet, "/tunnel/openflux/status", nil, &ps); err != nil {
		return fmt.Errorf("cannot ask the panel about the core: %w", err)
	}
	if !force && ps.BinaryExists && u.st.SHA256 == want {
		return nil
	}
	if ps.BinaryExists {
		// The panel stops the exits that use the file and starts them again on its next pass.
		if err := u.api.Do(ctx, http.MethodPost, "/tunnel/openflux/deleteBinary", map[string]string{}, nil); err != nil {
			return fmt.Errorf("cannot remove the old core: %w", err)
		}
	}
	body := map[string]string{"url": u.build.url(u.arch), "sha256": want}
	if err := u.api.Do(ctx, http.MethodPost, "/tunnel/openflux/download", body, nil); err != nil {
		return fmt.Errorf("the panel could not download the core: %w", err)
	}
	if err := u.api.Do(ctx, http.MethodGet, "/tunnel/openflux/status", nil, &ps); err != nil || !ps.BinaryExists {
		return errors.New("the core was downloaded but the panel does not show it")
	}
	u.st.Version, u.st.SHA256, u.st.InstalledAt = u.build.Version, want, u.now().UTC()
	return nil
}

// run checks at start and every few hours; a failed check is tried again sooner.
func (u *updater) run(ctx context.Context) {
	wait := firstCheck
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		cctx, cancel := context.WithTimeout(ctx, callBudget)
		err := u.ensure(cctx, false)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "better-openflux: %v\n", err)
			wait = retryAfter
		} else {
			wait = checkEvery
		}
	}
}
