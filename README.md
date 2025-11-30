# XELIS Mining Proxy

An open-source, high-performance XELIS mining proxy.
Designed to split the work between multiple miners.

This can also be used to connect miners that only support the stratum protocol to a getwork-only pool or daemon.

## Usage

- Compile or download xelis-mining-proxy and extract it, then run it
- Edit config.json for using a custom daemon or pool URL
- Start your miner of choice and point it to `127.0.0.1:5209` for stratum protocol, or `127.0.0.1:5210` for the getwork protocol.

## Command-line flags

- `--wallet <WALLET ADDRESS>`: Starts xelis-mining-proxy with the given wallet address
- `--debug`: Starts in debug mode

## Building from source

- Install Go
- Build using `go build .`
- Run the resulting binary