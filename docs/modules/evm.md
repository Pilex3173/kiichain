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
- Developers interact with the EVM module using Ethereum-compatible JSON-RPC endpoints and Cosmos SDK messages such as `MsgEthereumTx`. Contract deployment and calls follow standard Ethereum procedures.


## References
- [Evmos EVM Module](https://docs.evmos.org/)
- [Ethereum EVM Overview](https://ethereum.org/en/developers/docs/evm/)
- Referenced in `go.mod`
