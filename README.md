<img align="right" width="150" height="150" top="100" src="./public/op-gopher.png">

# `op-challenger` • [![ci](https://github.com/refcell/op-challenger/actions/workflows/ci.yml/badge.svg?label=ci)](https://github.com/refcell/op-challenger/actions/workflows/ci.yml) ![maintainer](https://img.shields.io/badge/maintainer-refcell-orange.svg?label=maintainer)

A multi-mode op-stack challenge agent for output dispute games written in golang.

> **Warning**
>
> This is a WIP Proof of Concept and is not ready, nor intended, for production use.
> Please visit the [Optimism Monorepo](https://github.com/ethereum-optimism/optimism) repository so you don't footgun yourself.


## Usage

The easiest way to install `op-challenger` is to download the latest binary from [challenger.refcell.org](https://challenger.refcell.org). To do this, just run:

```bash
curl -s https://challenger.refcell.org | sh
```

<p align="center">
  <img width="300px" src="./public/op-gopher.jpeg" />
</p>

Alternatively, you can build the `op-challenger` binary locally using the pre-configured makefile target by running `make build`, and then running `./op-challenger --help` to see the available options.

`op-challenger` is configurable via command line flags and environment variables. The help menu shows the available config options and can be accessed by running `./op-challenger --help`.

```bash
TODO: fillout
```


## Specifications

The op-challenger is a challenge agent for the output dispute game. It is responsible for challenging invalid state roots and invalid transactions. It is also responsible for challenging invalid fraud proofs.

This implementation is loosely based off the `op-proposer`, a proposer agent that is responsible for proposing new outputs to the `L2OutputOracle`. And this should make sense. Where the `op-proposer` _posts_ `output`s to the `L2OutputOracle`, the `op-challenger` validates these outputs and disputes them if invalid. Thus, the only additional functionality the `op-challenger` needs to implement is the ability to challenge invalid fraud proofs.

The naive challenge agent will be an attestation challenger which is a permissioned set of actors running the `op-challenger` attest.

The next iteration of the challenge agent will use fault proofs to challenge invalid outputs. This will involve a more complex dispute game which will allow for permissionless challengers by attaching bonds to each level of the dispute game.

A future iteration of permissionless outputs will allow for validity (zero-knowledge) proofs.

## Contributing

All contributions to magi are welcome. Before opening a PR, please submit an issue detailing the bug or feature. When opening a PR, please ensure that your contribution builds using the configured build target `make build`, has been linted with `golangci-lint` (simply run `make lint`), and contains tests when applicable.

## Acks

- [op-challenger](https://github.com/clabby/op-challenger): a challenge agent built in pure rust 🦀 by [clabby](https://github.com/clabby).
- [optimism](https://github.com/ethereum-optimism/optimism)
- [op-stack](https://stack.optimism.io/)
