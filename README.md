# The Hammer of the Gods crates.io Proxy

[![CI Status](https://github.com/hotg-ai/crates.io-proxy/actions/workflows/main.yml/badge.svg)](https://github.com/hotg-ai/crates.io-proxy/actions/workflows/main.yml)

[![Sync With Upstream](https://github.com/hotg-ai/crates.io-index/actions/workflows/sync.yml/badge.svg)](https://github.com/hotg-ai/crates.io-index/actions/workflows/sync.yml)

A simple mirror that proxy requests to crates.io and cache any crates that are
downloaded.

## Getting Started

TL;DR: Add the following to your `~/.cargo/config.toml`:

```toml
# Define a source for our mirror that points to hotg's index.
[source.mirror]
registry = "https://github.com/hotg-ai/crates.io-index-mirror"

# The crates.io default source for crates is available under the name
# "crates-io". We can use the "replace-with" key to override it with our mirror.
[source.crates-io]
replace-with = "mirror"
```

See Cargo's documentation on [*Source Replacement*][source-replacement] for a
more detailed explanation of how it works.

## Running Your Own Proxy

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
