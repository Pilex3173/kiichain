# ERC20 Module

The **ERC20 module** provides compatibility for the Ethereum ERC20 token standard within KiiChain. It enables token minting, transfer, and conversion between Cosmos native assets and ERC20 tokens.

## Overview
This module bridges Cosmos and EVM token models, allowing both ERC20 and Cosmos-based applications to operate seamlessly within the same chain.

## Key Features
- Supports ERC20 token mint, burn, and transfer.
- Converts tokens between EVM and Cosmos formats.
- Ensures 1:1 token representation across both environments.
- Integrates with bank and EVM modules.

## Main Functions
- **ConvertERC20** — Converts ERC20 tokens to native format.
- **ConvertCoin** — Converts Cosmos coins to ERC20 format.
- **RegisterTokenPair** — Registers cross-representation token pairs.

## References
- [Evmos ERC20 Module](https://docs.evmos.org/modules/erc20/)
- [ERC20 Standard](https://eips.ethereum.org/EIPS/eip-20)
- Referenced in `go.mod`
- 
