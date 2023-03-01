<p align="center">
    <img src="./assets/banner.png" width="500">
</p>

## Droid

Droid is a simple HTTP server that exposes some information about a Tendermint blockchain node.

It allows you to retrieve the node ID, public key, latest block, latest block height, and health status of the node.

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
- [Endpoints](#endpoints)
- [Configuration](#configuration)
- [Contribute](#contribute)

## Requirements

- Go 1.18 or later

## Installation

Clone the repository:

```bash
$ git clone https://github.com/osmosis-labs/droid.git
```

Build the binary:

```bash
make build
```

## Usage

Start the HTTP server:

```bash
make run
```

Open a web browser and go to http://localhost:8080/node_id to retrieve the node ID.

### Endpoints

- `/node_id`: Retrieve the node ID.
- `/pub_key`: Retrieve the public key of the node.
- `/block`: Retrieve the latest block.
- `/height`: Retrieve the latest block height.
- `/health`: Retrieve the health status of the node.

### Configuration

By default, the application will look for a `config.yaml` file in the current directory and in the `$HOME/.droid` directory. The file should contain the following structure:

```yaml
RPC_ENDPOINT: http://0.0.0.0:26657
LCD_ENDPOINT: http://0.0.0.0:1317
```

If the `config.yaml` file is not found, the application will create a default one with the above values.

Alternatively, you can specify the endpoint values using environment variables. To do so, set the `RPC_ENDPOINT` and `LCD_ENDPOINT` environment variables with the desired values before running the application. The environment variable values will take precedence over any values in the `config.yaml` file. If the file does not exist, a default one will be created.

## Contributing

Contributions are welcome! Please open a pull request or an issue.
