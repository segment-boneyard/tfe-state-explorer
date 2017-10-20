# tfe-state-explorer

Simple shell for exploring remote terraform enterprise state, with autocomplete.

## Building

```bash
$ go get github.com/kardianos/govendor
$ govendor sync
$ go build
```

## Running

```bash
$ ./tfe-state-explorer
```

Running requires you to have your ATLAS_TOKEN exported into the environment.
