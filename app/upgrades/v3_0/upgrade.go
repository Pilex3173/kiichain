package v300

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/kiichain/kiichain/v5/app/keepers"
	"github.com/kiichain/kiichain/v5/app/upgrades/utils"
	"github.com/kiichain/kiichain/v5/precompiles/oracle"
)

// CreateUpgradeHandler creates the upgrade handler for the v3.0.0 upgrade
// This install the new precompile into the precompiles list for the EVM module
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		// State the context and log
		ctx := sdk.UnwrapSDKContext(c)
		ctx.Logger().Info("Starting module migrations...")

		// Run the module migrations, it will start the new module with it's init genesis
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		// Install the new precompile
		err = utils.InstallNewPrecompiles(
			ctx,
			keepers,
			[]common.Address{
				common.HexToAddress(oracle.OraclePrecompileAddress),
			},
		)
		if err != nil {
			return vm, err
		}

		// Log the upgrade completion
		ctx.Logger().Info("Upgrade v3.0.0 complete")
		return vm, nil
	}
}
