# Feedreader

Make sure goose is installed

```bash
    go install github.com/pressly/goose/v3/cmd/goose@latest
```

run to get the goose variables for the migration set

```bash
    ./load_env.sh
```

to start with live reloads the app run

```bash
    air
```

To install template formatting, I've used

- https://www.npmjs.com/package/prettier-plugin-go-template

```bash
    pnpm init
```

## You could use pocketbase here instead of just a simple sqlite file

You could run pocketbase on the same host and you'd get ok performance. Will likely still out perform php style apps

See .data/tests

```shell
    cd ./tests && go test -bench=. -benchmem
```

```shell
    goos: darwin
    goarch: arm64
    pkg: main/data/tests
    cpu: Apple M1 Pro
    BenchmarkSQLiteFetch-10            99073             11739 ns/op             864 B/op         23 allocs/op
    BenchmarkPocketBaseFetch-10         8101            146234 ns/op            6565 B/op        111 allocs/op
    PASS
    ok      main/data/tests 2.676s
```

## To read

- https://daveceddia.com/implement-a-design-with-css/

## To do

might be nice to have the external pages load in an iframe. And then have a percentage read based upon the scroll depth.
Unsure as to how we would constrain the size of the page to avoid horizontal scroll bars

- https://github.com/samber/lo

https://data-star.dev/guide/datastar_expressions_javascript/#external-scripts

Add a date that we can sort on use go to parse the timestamps that come with
