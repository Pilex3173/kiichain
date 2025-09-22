// This file is based on the Cosmos EVM package
// The original implementation can be found at: `https://github.com/cosmos/evm/blob/main/ante/evm/mono_decorator.go`
// These are the main changes to the original implementation:
// - VerifyIfAccountExists has been moved up, this ensures that the account is created before the transaction is processed
// - After gas consumption, the fees are converted using the fee abstraction module
// - ConsumeFeesAndEmitEvent will now use the fee calculated by the fee abstraction module
// - VerifyAccountBalance will check if the user has enough balance to pay for the transaction value (before was fee + value)
// - The key ContextPaidFeesKey is defined on the context to store the paid fees, this is used to refund the gas under the evm module
//   - EVM module counterpart is defined under `x/vm/keeper/gas.go`

package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	evmante "github.com/cosmos/evm/ante/evm"
	anteinterfaces "github.com/cosmos/evm/ante/interfaces"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	antetypes "github.com/kiichain/kiichain/v5/ante/types"
)

// MonoDecorator is a single decorator that handles all the prechecks for
// ethereum transactions.
type MonoDecorator struct {
	accountKeeper        anteinterfaces.AccountKeeper
	feeMarketKeeper      anteinterfaces.FeeMarketKeeper
	evmKeeper            anteinterfaces.EVMKeeper
	feeAbstractionKeeper antetypes.FeeAbstractionKeeper
	maxGasWanted         uint64
}

// NewEVMMonoDecorator creates the 'mono' decorator, that is used to run the ante handle logic
// for EVM transactions on the chain.
//
// This runs all the default checks for EVM transactions enable through Cosmos EVM.
// Any partner chains can use this in their ante handler logic and build additional EVM
// decorators using the returned DecoratorUtils
func NewEVMMonoDecorator(
	accountKeeper anteinterfaces.AccountKeeper,
	feeMarketKeeper anteinterfaces.FeeMarketKeeper,
	evmKeeper anteinterfaces.EVMKeeper,
	feeAbstractionKeeper antetypes.FeeAbstractionKeeper,
	maxGasWanted uint64,
) MonoDecorator {
	return MonoDecorator{
		accountKeeper:        accountKeeper,
		feeMarketKeeper:      feeMarketKeeper,
		evmKeeper:            evmKeeper,
		feeAbstractionKeeper: feeAbstractionKeeper,
		maxGasWanted:         maxGasWanted,
	}
}

// AnteHandle handles the entire decorator chain using a mono decorator.
func (md MonoDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// 0. Basic validation of the transaction
	var txFeeInfo *txtypes.Fee
	if !ctx.IsReCheckTx() {
		// NOTE: txFeeInfo is associated with the Cosmos stack, not the EVM. For
		// this reason, the fee is represented in the original decimals and
		// should be converted later when used.
		txFeeInfo, err = evmante.ValidateTx(tx)
		if err != nil {
			return ctx, err
		}
	}

	evmDenom := evmtypes.GetEVMCoinDenom()

	// 1. setup ctx
	ctx, err = evmante.SetupContextAndResetTransientGas(ctx, tx, md.evmKeeper)
	if err != nil {
		return ctx, err
	}

	// 2. get utils
	decUtils, err := evmante.NewMonoDecoratorUtils(ctx, md.evmKeeper)
	if err != nil {
		return ctx, err
	}

	// NOTE: the protocol does not support multiple EVM messages currently so
	// this loop will complete after the first message.
	for i, msg := range tx.GetMsgs() {
		ethMsg, ethTx, err := evmtypes.UnpackEthMsg(msg)
		if err != nil {
			return ctx, err
		}

		feeAmt := ethMsg.GetFee()
		gas := ethTx.Gas()
		fee := sdkmath.LegacyNewDecFromBigInt(feeAmt)
		gasLimit := sdkmath.LegacyNewDecFromBigInt(new(big.Int).SetUint64(gas))

		// TODO: computation for mempool and global fee can be made using only
		// the price instead of the fee. This would save some computation.
		//
		// 2. mempool inclusion fee
		if ctx.IsCheckTx() && !simulate {
			// FIX: Mempool dec should be converted
			if err := evmante.CheckMempoolFee(fee, decUtils.MempoolMinGasPrice, gasLimit, decUtils.Rules.IsLondon); err != nil {
				return ctx, err
			}
		}

		if ethTx.Type() == ethtypes.DynamicFeeTxType && decUtils.BaseFee != nil {
			// If the base fee is not empty, we compute the effective gas price
			// according to current base fee price. The gas limit is specified
			// by the user, while the price is given by the minimum between the
			// max price paid for the entire tx, and the sum between the price
			// for the tip and the base fee.
			feeAmt = ethMsg.GetEffectiveFee(decUtils.BaseFee)
			fee = sdkmath.LegacyNewDecFromBigInt(feeAmt)
		}

		// 3. min gas price (global min fee)
		if err := evmante.CheckGlobalFee(fee, decUtils.GlobalMinGasPrice, gasLimit); err != nil {
			return ctx, err
		}

		// 4. validate msg contents
		if err := evmante.ValidateMsg(
			decUtils.EvmParams,
			ethTx,
		); err != nil {
			return ctx, err
		}

		// 5. signature verification
		if err := evmante.SignatureVerification(
			ethMsg,
			decUtils.Signer,
			decUtils.EvmParams.AllowUnprotectedTxs,
		); err != nil {
			return ctx, err
		}

		from := ethMsg.GetFrom()

		// Get the user account, this is used on the account verification process
		fromAddr := common.BytesToAddress(from)
		account := md.evmKeeper.GetAccount(ctx, fromAddr)
		if err := VerifyIfAccountExists(
			ctx,
			md.accountKeeper,
			account,
			fromAddr,
		); err != nil {
			return ctx, err
		}

		// 7. can transfer
		coreMsg := ethMsg.AsMessage(decUtils.BaseFee)

		// This checks if the user has enough balance to transfer the value (not the fees)
		if err := evmante.CanTransfer(
			ctx,
			md.evmKeeper,
			*coreMsg,
			decUtils.BaseFee,
			decUtils.EvmParams,
			decUtils.Rules.IsLondon,
		); err != nil {
			return ctx, err
		}

		// 8. gas consumption
		msgFees, err := evmkeeper.VerifyFee(
			ethTx,
			evmDenom,
			decUtils.BaseFee,
			decUtils.Rules.IsHomestead,
			decUtils.Rules.IsIstanbul,
			decUtils.Rules.IsShanghai,
			ctx.IsCheckTx(),
		)
		if err != nil {
			return ctx, err
		}

		// Here the fee abstraction module does it work
		// We check if the user has enough balance to pay for the fees using the
		// native token (evmDenom), if not we iterate the fee abstraction module tokens
		convertedMsgFees, err := md.feeAbstractionKeeper.ConvertNativeFee(ctx, from, msgFees)
		if err != nil {
			return ctx, err
		}

		// Here the gas is deducted from the user
		err = evmante.ConsumeFeesAndEmitEvent(
			ctx,
			md.evmKeeper,
			convertedMsgFees,
			from,
		)
		if err != nil {
			return ctx, err
		}

		// This checks if the user has enough balance
		// The main change here in comparison to the original implementation is that
		// we only check if the user has enough balance to pay for the transaction value
		// fees are ignored at this point and considered paid
		account = md.evmKeeper.GetAccount(ctx, fromAddr)
		if err := VerifyAccountBalance(
			ctx,
			md.accountKeeper,
			account,
			ethTx,
		); err != nil {
			return ctx, err
		}

		gasWanted := evmante.UpdateCumulativeGasWanted(
			ctx,
			gas,
			md.maxGasWanted,
			decUtils.GasWanted,
		)
		decUtils.GasWanted = gasWanted

		minPriority := evmante.GetMsgPriority(
			ethTx,
			decUtils.MinPriority,
			decUtils.BaseFee,
		)
		decUtils.MinPriority = minPriority

		// Update the fee to be paid for the tx adding the fee specified for the
		// current message.
		decUtils.TxFee.Add(decUtils.TxFee, ethMsg.GetFee())

		// Update the transaction gas limit adding the gas specified in the
		// current message.
		decUtils.TxGasLimit += gas

		// 9. increment sequence
		acc := md.accountKeeper.GetAccount(ctx, from)
		if acc == nil {
			// safety check: shouldn't happen
			return ctx, errorsmod.Wrapf(
				errortypes.ErrUnknownAddress,
				"account %s does not exist",
				from,
			)
		}

		if err := evmante.IncrementNonce(ctx, md.accountKeeper, acc, ethTx.Nonce()); err != nil {
			return ctx, err
		}

		// 10. gas wanted
		if err := evmante.CheckGasWanted(ctx, md.feeMarketKeeper, tx, decUtils.Rules.IsLondon); err != nil {
			return ctx, err
		}

		// 11. emit events
		txIdx := uint64(i)
		evmante.EmitTxHashEvent(ctx, ethMsg, decUtils.BlockTxIndex, txIdx)

		ctx.Logger().Info(
			"processed EVM message",
			"msg_index", txIdx,
			"from", from,
			"gas_wanted", decUtils.GasWanted,
			"gas_limit", gas,
			"fee", decUtils.TxFee,
			"min_priority", decUtils.MinPriority,
			"base_fee", decUtils.BaseFee,
			"tx_type", ethTx.Type(),
			"paid_fees", convertedMsgFees,
		)

		// Define the fee on the context for gas refunding
		ctx = ctx.WithValue(evmkeeper.ContextPaidFeesKey{}, convertedMsgFees)
	}

	if err := evmante.CheckTxFee(txFeeInfo, decUtils.TxFee, decUtils.TxGasLimit); err != nil {
		return ctx, err
	}

	ctx, err = evmante.CheckBlockGasLimit(ctx, decUtils.GasWanted, decUtils.MinPriority)
	if err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}
