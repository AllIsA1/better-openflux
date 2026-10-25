# Better OpenFlux

A plugin for the [LucX-UI](https://github.com/AlexeyLCP/lucx-ui) plugin system. It keeps the **OpenFlux exit program** of the panel
at a reviewed version: **OpenFlux v0.13.0**, which has more carriers (Yandex Volga, Board, Telemost, Avito besides Yandex.Docs,
Mail.ru and Cups.online), several links per carrier and the l3 exit mode.

## What it does

- The panel starts an OpenFlux inbound with a program file it keeps in its `bin` folder. The panel's own release ships an older
  build; this plugin makes the panel download the reviewed v0.13.0 build and check its hash. It is the panel that downloads
  (its Settings → Cores mechanism), over HTTPS from a release of this repository, and the download is refused unless the sha256
  in the plugin matches.
- It checks at start and every six hours. It replaces a core it did not install (the one that came with the panel) once, and leaves
  alone the one it installed, so a file you upload yourself on Settings → Cores stays until a newer plugin version brings a newer
  core. (The plugin has no page of its own; restarting it checks again.)
- Exits that use the file are stopped when it is replaced and started again by the panel on its next pass.
- Switch the plugin off or remove it and the core stays as it is.

The inbound form (the new carriers, several links per carrier, the l3/l4 choice) is part of the panel build `…-plugins.8` and
later; the plugin only brings the program those fields need.

## Permissions

`api:openflux:read` (is a core there) and `api:openflux:write` (download a core by a link and a hash, remove it) — the narrow
permissions of the plugin system for exactly these panel routes and no other. **Write is a dangerous permission**: the panel runs
this program as root, so the operator confirms it when installing.

## The core

`core.go` pins the version, the upstream commit and the sha256 of each architecture. The binaries are built by
`.github/workflows/core.yml` (`scripts/build-core.sh`: the upstream tag must point at exactly the named commit; static, CGO off,
`-trimpath`) and published as the release `core-<version>-<n>` of this repository. OpenFlux is GPL-3.0-or-later software of its
authors (https://github.com/p1neappleXpress/OpenFlux); the release names the source commit and builds it unchanged.

## Build

`make build` writes `bin/<arch>/betteropenflux` (commit `bin/`: the panel downloads the repository as it is). `make test` runs the tests.

MIT, see `LICENSE`. The OpenFlux program keeps its own license.
