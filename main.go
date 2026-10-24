// Copyright (c) 2026 AllIsA1.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"
)

func main() {
	var api panelAPI
	if p, err := NewPanel(); err != nil {
		log.Printf("better-openflux: %v; the core is left as it is", err)
		api = failingPanel{err}
	} else {
		api = p
	}
	u := newUpdater(api, runtime.GOARCH, filepath.Join(StateDir(), "state.json"), pinned)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go u.run(ctx)
	log.Printf("better-openflux plugin ready (core %s)", pinned.Version)
	if err := Serve(newMux(u)); err != nil {
		log.Fatal(err)
	}
}

type failingPanel struct{ err error }

func (f failingPanel) Do(context.Context, string, string, any, any) error { return f.err }
