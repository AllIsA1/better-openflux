# Better OpenFlux

A plugin for the [LucX-UI](https://github.com/AlexeyLCP/lucx-ui) plugin system. **It does nothing now.**

It used to make the panel download the reviewed OpenFlux v0.13.0 exit program (more carriers, several links per carrier, l3) because the
panel's own release shipped an older build. Since **lucx.326** ([#148](https://github.com/AlexeyLCP/lucx-ui/issues/148)) the panel's
release brings OpenFlux v0.13.0 itself, so the plugin has no job left (version 2.0.0 removed it: no core download, no page, no
permissions, no `requires`).

It is left installed for the time being and answers only the plugin system's health check. Remove it from the Plugins page whenever you like;
the OpenFlux core of the panel stays as it is. A panel older than lucx.326 needs Better OpenFlux **1.0.3**.

The `core-v0.13.0-1` release of this repository (the binaries the old versions installed) stays where it is.

MIT, see `LICENSE`.
