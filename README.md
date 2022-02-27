# The Hammer of the Gods crates.io Mirror

[![CI Status](https://github.com/hotg-ai/crates.io-mirror/actions/workflows/main.yml/badge.svg)](https://github.com/hotg-ai/crates.io-mirror/actions/workflows/main.yml)

[![Sync With Upstream](https://github.com/hotg-ai/crates.io-index/actions/workflows/sync.yml/badge.svg)](https://github.com/hotg-ai/crates.io-index/actions/workflows/sync.yml)

A mirror that proxies requests to crates.io and caches any crates that are
downloaded.

## Getting Started

TL;DR: Add the following to your `~/.cargo/config.toml`:

```toml
# Define a source for our mirror that points to hotg's index.
[source.mirror]
registry = "https://github.com/hotg-ai/crates.io-index"

# The crates.io default source for crates is available under the name
# "crates-io". We can use the "replace-with" key to override it with our mirror.
[source.crates-io]
replace-with = "mirror"
```

See Cargo's documentation on [*Source Replacement*][source-replacement] for a
more detailed explanation of how it works.

## Running Your Own Mirror

Cargo works by consulting a git repository (the "index") which contains metadata
for every version of every crate that has ever been published. This index also
contains a `config.json` which tells `cargo` things like the endpoints to use
when downloading a crate or querying the crates.io API.

That means making `cargo` download `*.crate` files from a different location
isn't as easy as spinning up a server and overriding an environment variable.
Instead, we need to create our own index which tracks the upstream and overrides
the download endpoint.

### Setting Up The crates.io Index

Setting up your own index comes in two steps,

1. Create your own copy of https://github.com/rust-lang/crates.io-index
2. Update `config.json` to point at your proxy server
3. Keep the index up to date

The first step is trivial - just fork [the official crates.io index][index].

Similarly, updating the `config.json` is just a case of opening the file in your
editor and making the required changes.

```console
$ git clone git@github.com:hotg-ai/crates.io-index.git
$ cd crates.io-index
$ cat config.json
```

Keeping the index up to date is a bit trickier.

Crates.io doesn't support webhooks for notifying subscribers when a new crate is
published ([`rust-lang/crates.io#381`][webhooks]), so we'll need to use GitHub
Actions to periodically poll the upstream index for updates and `rebase` our
changes on top of it.

For convenience, here is a GitHub Actions workflow that you can copy into
`.github/workflows/sync.yml`. Tweak as required.

```yml
name: Sync With Upstream

on:
  schedule:
    # Re-run every 5 minutes
    - cron: '*/5 * * * *'
  # Or on demand
  workflow_dispatch:

jobs:
  sync:
    runs-on: ubuntu-latest
    env:
      USER_NAME: 'Michael-F-Bryan'
      USER_EMAIL: 'michael@hotg.ai'
      UPSTREAM: 'https://github.com/rust-lang/crates.io-index'
    steps:
      - uses: actions/checkout@v2
        with:
          fetch-depth: 0
      - name: Set up Git
        run: |
          git remote add upstream ${{ env.UPSTREAM }}
          git config --global user.name ${{ env.USER_NAME }}
          git config --global user.email ${{ env.USER_EMAIL }}
      - name: Fetch Upstream
        run: git fetch upstream
      - name: Rebase
        run: git rebase upstream/master
      - name: Check for modified files
        id: git-check
        run: echo ::set-output name=modified::$(if [ "$(git rev-parse HEAD)" = "${{ github.sha }}" ]; then echo "false"; else echo "true"; fi)
      - name: Push Changes
        if: steps.git-check.outputs.modified == 'true'
        run: git push --force origin master
```

### Using The Server

Most of the mirror server's functionality is configured via either command-line
arguments or environment variables.

```
Usage:
  crates-io-mirror [OPTIONS]

Application Options:
  -v, --verbose    Show more verbose debug information [$VERBOSE]
  -u, --upstream=  The URL to proxy requests to (default: https://crates.io/) [$UPSTREAM]
  -H, --host=      The interface to listen on (default: localhost) [$HOST]
  -p, --port=      The port to use (default: 8000) [$PORT]
  -b, --bucket=    The bucket to cache responses in [$S3_BUCKET]
  -c, --cache-dir= The directory to use when caching locally (default: ~/.cache/crates.io-mirror) [$CACHE_DIR]

Help Options:
  -h, --help       Show this help message

exit status 1
```

The server itself is a simple Go program that is compiled and tested in the
normal way.

```console
$ go build .
$ go test -race .
```

This repository also contains a `Dockerfile` for deploying to production.

```console
$ docker build --tag=crates.io-proxy:latest --tag=crates.io-proxy:$(git rev-parse HEAD) .
```

### Caching

There are two main caching strategies, local caching and S3.

The mirror will prefer to use S3 whenever the name of an S3 bucket is provided,
using [the default AWS credential chain][aws-auth] for authentication.

Otherwise, files will be cached locally under the `crates.io-mirror/` folder in
the user's cache directory (e.g. `~/.cache/crates.io-mirror/`).

When the mirror runs for the first time, it won't have cached any crates and
will need to send a request upstream. The server logs will look something like
this:

```json
{"level":"info","ts":1645987370.3546326,"caller":"crates.io-mirror/main.go:42","msg":"Started","args":{"Verbose":false,"Upstream":"https://crates.io/","Host":"localhost","Port":8000,"Bucket":"","CacheDir":"~/.cache/crates.io-mirror"}}
{"level":"info","ts":1645987370.3548064,"caller":"crates.io-mirror/main.go:60","msg":"Serving","addr":"localhost:8000"}
{"level":"info","ts":1645987813.1638477,"caller":"crates.io-mirror/logging.go:42","msg":"Served a request","request-id":"b28f35ec-f19b-4c94-83d9-175866131b44","status-code":200,"status-text":"OK","bytes-written":25913,"url":"/api/v1/crates/async-trait/0.1.52/download","method":"GET","duration":0.897670102,"user-agent":"cargo 1.60.0-nightly (95bb3c92b 2022-01-18)","remote-addr":"127.0.0.1:52662"}
```

Cache hits will look something like this:

```json
{"level":"info","ts":1645987708.2469437,"caller":"crates.io-mirror/cache.go:25","msg":"Serving up a cached response","request-id":"f9c0dbe8-84d8-4dce-ad47-34c5c6e9cf81","bytes":25913,"path":"/api/v1/crates/async-trait/0.1.52/download"}
{"level":"info","ts":1645987708.2469811,"caller":"crates.io-mirror/logging.go:42","msg":"Served a request","request-id":"f9c0dbe8-84d8-4dce-ad47-34c5c6e9cf81","status-code":200,"status-text":"OK","bytes-written":25913,"url":"/api/v1/crates/async-trait/0.1.52/download","method":"GET","duration":0.00008054,"user-agent":"cargo 1.60.0-nightly (95bb3c92b 2022-01-18)","remote-addr":"127.0.0.1:52658"}
```



## License

This project is licensed under either of

 * Apache License, Version 2.0, ([LICENSE-APACHE](LICENSE-APACHE.md) or
   http://www.apache.org/licenses/LICENSE-2.0)
 * MIT license ([LICENSE-MIT](LICENSE-MIT.md) or
   http://opensource.org/licenses/MIT)

at your option.

### Contribution

Unless you explicitly state otherwise, any contribution intentionally
submitted for inclusion in the work by you, as defined in the Apache-2.0
license, shall be dual licensed as above, without any additional terms or
conditions.

[source-replacement]: https://doc.rust-lang.org/cargo/reference/source-replacement.html
[index]: https://github.com/rust-lang/crates.io-index
[webhooks]: https://github.com/rust-lang/crates.io/issues/381
[aws-auth]: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
