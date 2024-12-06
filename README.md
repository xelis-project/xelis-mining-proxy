# xelis-mining-proxy v0.1.1
An open-source, high-performance XELIS mining proxy.
Designed to split the work between multiple miners.

## Usage
1. Download xelis-mining-proxy and extract it, then run it
2. Start your miner of choice and point it to the proxy's `IP:PORT`.

Example xelis-mining-proxy mining with xelis_miner:
```
./xelis_miner -m YOUR_WALLET_ADDRESS --daemon-address 127.0.0.1:5210
```

## Command-line flags
- `--wallet <WALLET ADDRESS>`: Starts xelis-mining-proxy with the given wallet address
- `--debug`: Starts in debug mode