# EVM Module

The **EVM module** brings full Ethereum Virtual Machine (EVM) compatibility to KiiChain, enabling the execution of Solidity-based smart contracts directly on the network.

## Overview
By integrating EVM functionality, KiiChain allows developers to deploy, call, and interact with Ethereum-style contracts while benefiting from Cosmos SDK’s modular and efficient architecture.

## Key Features
- Supports Solidity smart contracts.
- Compatible with Ethereum development tools (Remix, MetaMask, Hardhat).
- Enables interoperability between EVM and native Cosmos modules.
- Provides a gas model aligned with the Cosmos SDK fee system.

## Main Functions
- **DeployContract** — Upload and initialize Solidity bytecode.
- **CallContract** — Execute functions from existing contracts.
- **EstimateGas** — Calculate gas cost for transactions.
- **QueryState** — Inspect contract storage and execution traces.

## References
- [Evmos EVM Module](https://docs.evmos.org/)
- [Ethereum EVM Overview](https://ethereum.org/en/developers/docs/evm/)
- Referenced in `go.mod`
.