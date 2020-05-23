# gusano

Package-wide static analysis of GO code

`gusano` is a framework for implementing static analysis on GO packages.
`gusano` is a fork of the invaluable [revive](https://github.com/mgechev/revive) linter but it allows developing analysis to cope with whole packages. This, for example, makes possible to implement analysis like unused symbols (var, const, types, funcs, ...) detection.

The main purpose of `gusano` is to serve as sandbox for evolutions in `revive`.

# How to use `gusano`

```bash
$ cd your/go/module/root
$ gusano ./...
```

/!\ command line flags, except `-formatter`, do not work
