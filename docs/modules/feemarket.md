# FeeMarket Module

The **FeeMarket module** introduces a dynamic gas fee mechanism inspired by Ethereum’s EIP-1559, enabling better transaction prioritization and fee predictability.

## Overview
This module provides a base fee adjustment mechanism that balances demand and supply for block space, improving user experience and preventing spam.

## Key Features
- Implements base fee adjustment per block.
- Supports priority fees (tips) for faster inclusion.
- Integrates with the EVM module’s gas pricing.
- Ensures predictable transaction costs.

## Main Functions
- **GetBaseFee** — Retrieves current base fee.
- **CalculateNextBaseFee** — Determines base fee for the next block.
- **SetBaseFee** — Updates fee parameters in governance.

## References
- [EIP-1559 Specification](https://eips.ethereum.org/EIPS/eip-1559)
- [Evmos FeeMarket Module](https://docs.evmos.org/modules/feemarket/)
- Referenced in `go.mod`
