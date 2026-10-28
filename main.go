// Copyright (c) 2026 AllIsA1.
// SPDX-License-Identifier: MIT

package main

import (
	"log"
	"net/http"
)

// Since lucx.326 the panel ships OpenFlux v0.13.0 itself, so there is nothing left for this plugin to do.
// It only answers the plugin system's health check until it is removed.
func main() {
	log.Print("better-openflux plugin ready (nothing to do: the panel brings OpenFlux v0.13.0 itself)")
	if err := Serve(http.NotFoundHandler()); err != nil {
		log.Fatal(err)
	}
}
