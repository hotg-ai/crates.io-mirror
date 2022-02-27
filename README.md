# HOTG's crates.io Proxy

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

This proxy work using Cargo's builtin [*Source Replacement*][source-replacement]
feature.

## Setting Up Your Own Registry

## Deploying the Proxy

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
