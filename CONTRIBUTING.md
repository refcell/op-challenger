# Contributing

Thank you for looking to contribute!

We welcome all contributions! Before opening a PR, please submit an issue detailing the bug or feature. When opening a PR, please ensure that your contribution builds using the configured build target `make build`, has been linted with `golangci-lint` (simply run `make lint`), and contains tests when applicable.

## Dependencies

To work in this repo, you'll need to install:

1. [Golang](https://go.dev/)
1. [Docker](https://docs.docker.com/get-docker/)

And clone the [Optimism Monorepo](https://github.com/ethereum-optimism/optimism)

## Quickstart

1. Clone the repo
```sh
git clone git@github.com:refcell/op-challenger.git
```

2. Configure your dev environment
```sh
# Set the MONOREPO_DIR variable
nvim .env.devnet
# Set up your env vars
source .env.devnet
# On the L1 service, port forward the websocket endpoint port (8546)
nvim $MONOREPO_DIR/ops-bedrock/docker-compose.yml
# Install forge deps
(cd ./testdata/mock-dgf && forge install)
# Start the devnet and deploy the mock dispute game factory
./start_devnet.sh
```
3. Start the `op-challenger`.
```sh
make run
```

## Linting

To lint your code, run:
```sh
make lint
```

You can also run `make format` to format your code and fix most linting errors.