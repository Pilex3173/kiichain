package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/kiichain/kiichain/v5/x/feeabstraction/types"
)

// TestMsgUpdateParamsValidate tests the Validate method of MsgUpdateParams
func TestMsgUpdateParamsValidate(t *testing.T) {
	// Prepare all the test cases
	testCases := []struct {
		name        string
		msg         *types.MsgUpdateParams
		errContains string
	}{
		{
			name: "valid - default params",
			msg: types.NewMessageUpdateParams(
				authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				types.DefaultParams(),
			),
		},
		{
			name: "valid - custom params",
			msg: types.NewMessageUpdateParams(
				authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				types.NewParams("coin", "coinoracle", types.DefaultClampFactor, types.DefaultFallbackNativePrice, types.DefaultTwapLookbackWindow, true),
			),
		},
		{
			name:        "invalid - empty authority",
			msg:         types.NewMessageUpdateParams("", types.DefaultParams()),
			errContains: "empty address string is not allowed",
		},
		{
			name: "invalid - bad params",
			msg: types.NewMessageUpdateParams(
				authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				types.NewParams("", "coinoracle", types.DefaultClampFactor, math.LegacyZeroDec(), 0, true),
			),
			errContains: "native denom is invalid: invalid fee abstraction params",
		},
	}

	// Iterate through the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Validate()

			// Check the error
			if tc.errContains == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}

// TestMsgUpdateFeeTokensValidate tests the Validate method of MsgUpdateFeeTokens
func TestMsgUpdateFeeTokensValidate(t *testing.T) {
	// Prepare all the test cases
	testCases := []struct {
		name        string
		msg         *types.MsgUpdateFeeTokens
		errContains string
	}{
		{
			name: "valid - empty fee tokens",
			msg: types.NewMessageUpdateFeeTokens(
				authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				*types.NewFeeTokenMetadataCollection(),
			),
		},
		{
			name: "valid - fee tokens",
			msg: types.NewMessageUpdateFeeTokens(
				authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				*types.NewFeeTokenMetadataCollection(
					types.NewFeeTokenMetadata("coin", "oracleCoin", 6, math.LegacyMustNewDecFromStr("0.01")),
				),
			),
		},
		{
			name: "invalid - empty authority",
			msg: types.NewMessageUpdateFeeTokens("",
				*types.NewFeeTokenMetadataCollection(
					types.NewFeeTokenMetadata("coin", "oracleCoin", 6, math.LegacyMustNewDecFromStr("0.01")),
				),
			),
			errContains: "empty address string is not allowed",
		},
		{
			name: "invalid - duplicate fee tokens",
			msg: types.NewMessageUpdateFeeTokens(
				authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				*types.NewFeeTokenMetadataCollection(
					types.NewFeeTokenMetadata("coin", "oracleCoin", 6, math.LegacyMustNewDecFromStr("0.01")),
					types.NewFeeTokenMetadata("coin", "oracleCoin", 6, math.LegacyMustNewDecFromStr("0.01")),
				),
			),
			errContains: "duplicate denom found: coin: invalid fee token metadata",
		},
	}

	// Iterate through the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Validate()

			// Check the error
			if tc.errContains == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}
