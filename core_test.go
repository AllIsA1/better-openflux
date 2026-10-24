// Copyright (c) 2026 AllIsA1.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testBuild = coreBuild{
	Version: "v0.13.0", Commit: strings64("c", 40), Release: "core-v0.13.0-1",
	SHA256: map[string]string{"amd64": strings64("a", 64), "arm64": strings64("b", 64)},
}

func strings64(s string, n int) string { return strings.Repeat(s, n) }

type call struct {
	method, path string
	body         map[string]string
}

// fakePanel plays the panel's OpenFlux core routes.
type fakePanel struct {
	exists  bool
	running bool
	calls   []call
	fail    map[string]error // by path
}

func (p *fakePanel) Do(_ context.Context, method, path string, in, out any) error {
	c := call{method: method, path: path}
	if m, ok := in.(map[string]string); ok {
		c.body = m
	}
	p.calls = append(p.calls, c)
	if err := p.fail[path]; err != nil {
		return err
	}
	switch path {
	case "/tunnel/openflux/status":
		raw, _ := json.Marshal(map[string]any{"binaryExists": p.exists, "binaryPath": "/x/bin/openflux-linux-amd64", "probe": map[string]bool{"running": p.running}})
		return json.Unmarshal(raw, out)
	case "/tunnel/openflux/deleteBinary":
		p.exists, p.running = false, false
	case "/tunnel/openflux/download":
		p.exists = true
	default:
		return errors.New("the plugin asked for " + path + ", which it has no business with")
	}
	return nil
}

func (p *fakePanel) paths() []string {
	var out []string
	for _, c := range p.calls {
		if c.path != "/tunnel/openflux/status" {
			out = append(out, c.method+" "+c.path)
		}
	}
	return out
}

func newTestUpdater(t *testing.T, p *fakePanel) *updater {
	t.Helper()
	u := newUpdater(p, "amd64", filepath.Join(t.TempDir(), "state.json"), testBuild)
	u.now = func() time.Time { return time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC) }
	return u
}

func TestInstallsThePinnedCoreWhereThePanelHasNone(t *testing.T) {
	p := &fakePanel{}
	u := newTestUpdater(t, p)
	if err := u.ensure(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if got := p.paths(); len(got) != 1 || got[0] != "POST /tunnel/openflux/download" {
		t.Fatalf("calls = %v: with no core there is nothing to delete", got)
	}
	var dl call
	for _, c := range p.calls {
		if c.path == "/tunnel/openflux/download" {
			dl = c
		}
	}
	if dl.body["url"] != "https://github.com/AllIsA1/better-openflux/releases/download/core-v0.13.0-1/openflux-linux-amd64" || dl.body["sha256"] != testBuild.SHA256["amd64"] {
		t.Errorf("download = %v", dl.body)
	}
	if u.st.Version != "v0.13.0" || u.st.SHA256 != testBuild.SHA256["amd64"] || u.st.LastError != "" || u.st.InstalledAt.IsZero() {
		t.Errorf("state = %+v", u.st)
	}
}

func TestReplacesACoreItDidNotInstall(t *testing.T) {
	p := &fakePanel{exists: true, running: true} // the one the panel release shipped
	u := newTestUpdater(t, p)
	if err := u.ensure(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(p.paths(), ","); got != "POST /tunnel/openflux/deleteBinary,POST /tunnel/openflux/download" {
		t.Errorf("calls = %s: remove, then download", got)
	}
}

func TestLeavesAlonetheCoreItInstalledAndWhatTheOperatorPutThereSince(t *testing.T) {
	p := &fakePanel{}
	u := newTestUpdater(t, p)
	if err := u.ensure(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	p.calls = nil
	if err := u.ensure(context.Background(), false); err != nil || len(p.paths()) != 0 {
		t.Fatalf("a second check changed something: %v, %v", err, p.paths())
	}
	// ...also a file uploaded by hand on Settings, Cores: the plugin cannot tell and does not touch it
	p.running = true
	if err := u.ensure(context.Background(), false); err != nil || len(p.paths()) != 0 {
		t.Fatalf("an unchanged state was touched: %v, %v", err, p.paths())
	}
}

func TestInstallsAgainWhenThePanelLostTheCore(t *testing.T) {
	p := &fakePanel{}
	u := newTestUpdater(t, p)
	_ = u.ensure(context.Background(), false)
	p.exists = false // the panel's own update replaced the whole bin folder
	p.calls = nil
	if err := u.ensure(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if got := p.paths(); len(got) != 1 || got[0] != "POST /tunnel/openflux/download" {
		t.Errorf("calls = %v", got)
	}
}

func TestANewerPinnedBuildReplacesTheOldOne(t *testing.T) {
	p := &fakePanel{}
	u := newTestUpdater(t, p)
	_ = u.ensure(context.Background(), false)
	u.build.Version = "v0.14.0"
	u.build.Release = "core-v0.14.0-1"
	u.build.SHA256 = map[string]string{"amd64": strings64("d", 64), "arm64": strings64("e", 64)}
	p.calls = nil
	if err := u.ensure(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(p.paths(), ","); got != "POST /tunnel/openflux/deleteBinary,POST /tunnel/openflux/download" {
		t.Errorf("calls = %s", got)
	}
	if u.st.Version != "v0.14.0" {
		t.Errorf("state = %+v", u.st)
	}
}

func TestForceInstallsEvenWhenCurrent(t *testing.T) {
	p := &fakePanel{}
	u := newTestUpdater(t, p)
	_ = u.ensure(context.Background(), false)
	p.calls = nil
	if err := u.ensure(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if len(p.paths()) != 2 {
		t.Errorf("calls = %v", p.paths())
	}
}

func TestAFailedDownloadIsRememberedAndNotCountedAsInstalled(t *testing.T) {
	p := &fakePanel{fail: map[string]error{"/tunnel/openflux/download": errors.New("tunnel: sha256 mismatch")}}
	u := newTestUpdater(t, p)
	err := u.ensure(context.Background(), false)
	if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("err = %v", err)
	}
	if u.st.SHA256 != "" || !strings.Contains(u.st.LastError, "sha256 mismatch") {
		t.Errorf("state = %+v", u.st)
	}
	// a restart reads the problem back
	again := newUpdater(p, "amd64", u.path, testBuild)
	if !strings.Contains(again.st.LastError, "sha256 mismatch") {
		t.Errorf("state after restart = %+v", again.st)
	}
	// and the next try goes through once the panel works
	p.fail = nil
	p.calls = nil
	if err := again.ensure(context.Background(), false); err != nil || again.st.LastError != "" {
		t.Errorf("retry: %v, state %+v", err, again.st)
	}
}

func TestWithoutAPanelAnswerNothingIsChanged(t *testing.T) {
	p := &fakePanel{fail: map[string]error{"/tunnel/openflux/status": errors.New("panel answered HTTP 403")}}
	u := newTestUpdater(t, p)
	if err := u.ensure(context.Background(), false); err == nil {
		t.Error("a forbidden status call was not reported")
	}
	if got := p.paths(); len(got) != 0 {
		t.Errorf("calls = %v: it must not change anything blind", got)
	}
}

func TestRefusesAnArchitectureNothingIsPinnedFor(t *testing.T) {
	p := &fakePanel{}
	u := newUpdater(p, "riscv64", filepath.Join(t.TempDir(), "s.json"), testBuild)
	if err := u.ensure(context.Background(), false); err == nil || len(p.calls) != 0 {
		t.Errorf("err = %v, calls = %v", err, p.calls)
	}
}

func TestSnapshotSaysWhetherThePanelHasThePluginsCore(t *testing.T) {
	p := &fakePanel{exists: true}
	u := newTestUpdater(t, p)
	if s := u.snapshot(context.Background()); s.Current || !s.Present || s.Want != "v0.13.0" {
		t.Errorf("before: %+v", s)
	}
	_ = u.ensure(context.Background(), false)
	p.running = true
	if s := u.snapshot(context.Background()); !s.Current || !s.Present || !s.Running || s.Installed != "v0.13.0" {
		t.Errorf("after: %+v", s)
	}
	p.exists = false
	if s := u.snapshot(context.Background()); s.Current || s.Present {
		t.Errorf("lost: %+v", s)
	}
}

func TestThePinnedBuildIsWellFormed(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		h := pinned.SHA256[arch]
		if len(h) != 64 || strings.Trim(h, "0123456789abcdef") != "" {
			t.Errorf("%s hash %q", arch, h)
		}
		want := "https://github.com/AllIsA1/better-openflux/releases/download/" + pinned.Release + "/openflux-linux-" + arch
		if pinned.url(arch) != want {
			t.Errorf("url = %s", pinned.url(arch))
		}
	}
	if pinned.SHA256["amd64"] == pinned.SHA256["arm64"] || len(pinned.Commit) != 40 || !strings.Contains(pinned.Release, pinned.Version) {
		t.Errorf("pinned = %+v", pinned)
	}
	raw, err := os.ReadFile("lucx-plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, p := range m.Permissions {
		if strings.HasPrefix(p, "api:") && !strings.HasPrefix(p, "api:openflux:") {
			t.Errorf("the plugin asks for %s, which it does not need", p)
		}
	}
}

func TestPageAndCallsAreServed(t *testing.T) {
	p := &fakePanel{exists: true}
	u := newTestUpdater(t, p)
	srv := httptest.NewServer(newMux(u))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/ui/index.html")
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("page: %v %v", err, res)
	}
	var s snapshot
	res, _ = http.Get(srv.URL + "/api/status")
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil || s.Want != "v0.13.0" || !s.Present {
		t.Fatalf("status: %v %+v", err, s)
	}
	res, _ = http.Post(srv.URL+"/api/install", "application/json", strings.NewReader("{}"))
	if res.StatusCode != 200 {
		t.Fatalf("install: %d", res.StatusCode)
	}
	if got := strings.Join(p.paths(), ","); got != "POST /tunnel/openflux/deleteBinary,POST /tunnel/openflux/download" {
		t.Errorf("calls = %s", got)
	}
	p.fail = map[string]error{"/tunnel/openflux/download": errors.New("no")}
	res, _ = http.Post(srv.URL+"/api/install", "application/json", strings.NewReader("{}"))
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("a failed install answered %d", res.StatusCode)
	}
	if res, _ := http.Get(srv.URL + "/ui/../main.go"); res.StatusCode == 200 {
		t.Error("a file outside ui/ was served")
	}
	if res, _ := http.Get(srv.URL + "/api/other"); res.StatusCode != 404 {
		t.Errorf("unknown call answered %d", res.StatusCode)
	}
}
