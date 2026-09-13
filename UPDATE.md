# Updating mrw

Install from [GitHub Latest](https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases/latest),
not from a leftover on PATH.

`mrw version` must print the **tag** and the **SHA that tag cut from**. At
v1.15.0 that is `v1.15.0 (31422d8)`. If it prints an older SHA, or
`dev (…)`, you are running a different binary than the release — usually one
earlier on PATH, or a working-tree build.

Do not install as a side quest from an old PATH copy. See which binary you
actually invoke, then replace **that** file:

```sh
command -v mrw
mrw version
```

Overwrite it with the Latest asset for your OS/arch (see the README Install
recipe), or `go build` from a checkout at the tag. Archives and
`SHA256SUMS.txt` sit next to the raw binaries on the same release.

The changelog is the [release notes](https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases).
This file is only how to update.
