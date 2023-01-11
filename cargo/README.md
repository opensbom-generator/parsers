# Cargo Parser

![image](https://user-images.githubusercontent.com/3935899/202314883-9331600e-cc0d-442e-9015-c244eee0518a.png)

This directory contains the cargo project parser. It extracts information about
a Rust project that handles its dependencies using cargo.

## How Data Is Collected

Dependency data is read from three different sources: Reading the `Cargo.toml` 
config file, reading the `Cargo.lock` file and calling cargo to extract metadata.

The main chunk of information comes from running the `cargo metadata` subcommand
and parsing its ouput. Data for the SBOM is augmented from the Cargo lockfile 
the main one being the computed crate hashes reused in the `metadata.Package`s
containing dependency data.

## Tests

The parser includes a full test suite. To run the parser's tests simply run:

```
 go test ./cargo/...

```
Integration tests are built by using a faked parser implementation. When adding
functions to the implementation interface, regenerate the cargo fakes by running:

```
 go generate ./cargo/...
```

## Known Issues

* Cargo metadata is currently hardcoded to Linux as the parsers project does not
yet have awarness of the platform executing the parser ([Issue #25](https://github.com/opensbom-generator/parsers/issues/25))
