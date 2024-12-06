# xelis-mining-proxy v1.0.0
An open-source, high-performance XELIS mining proxy.
Designed to split the work between multiple miners.

## Usage
1. Download xelis-mining-proxy and extract it, then run it
2. Edit config.json for using a custom daemon or pool URL
3. Start your miner of choice and point it to `127.0.0.1:5209`

## Command-line flags
- `--wallet <WALLET ADDRESS>`: Starts xelis-mining-proxy with the given wallet address
- `--debug`: Starts in debug mode

## Building from source
1. Install Go
2. Build using `go build .`

### Cross-compiling
1. Install Go
2. Install XGO: `go get src.techknowlogick.com/xgo`
3. Run the `build.sh` script