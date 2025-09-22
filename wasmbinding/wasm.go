package wasmbinding

import (
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	evmkeeper "github.com/cosmos/evm/x/vm/keeper"

	"github.com/kiichain/kiichain/v5/wasmbinding/bech32"
	evmwasmbinding "github.com/kiichain/kiichain/v5/wasmbinding/evm"
	"github.com/kiichain/kiichain/v5/wasmbinding/oracle"
	tfbinding "github.com/kiichain/kiichain/v5/wasmbinding/tokenfactory"
	oraclekeeper "github.com/kiichain/kiichain/v5/x/oracle/keeper"
	tokenfactorykeeper "github.com/kiichain/kiichain/v5/x/tokenfactory/keeper"
)

// RegisterCustomPlugins registers custom plugins for the wasm module
func RegisterCustomPlugins(
	bank bankkeeper.Keeper,
	tokenFactory *tokenfactorykeeper.Keeper,
	evmKeeper *evmkeeper.Keeper,
	oracleKeeper oraclekeeper.Keeper,
) []wasmkeeper.Option {
	// Register custom query plugins
	tokenFactoryQueryPlugin := tfbinding.NewQueryPlugin(bank, tokenFactory)
	evmQueryPlugin := evmwasmbinding.NewQueryPlugin(evmKeeper)
	bech32QueryPlugin := bech32.NewQueryPlugin()
	oracleQueryPlugin := oracle.NewQueryPlugin(oracleKeeper)

	// Create the central query plugin
	queryPlugin := NewQueryPlugin(
		tokenFactoryQueryPlugin,
		evmQueryPlugin,
		bech32QueryPlugin,
		oracleQueryPlugin,
	)

	// Register custom message handler decorators
	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(queryPlugin),
	})

	// Create the custom messenger to the token factory
	tokenFactoryMessenger := tfbinding.NewCustomMessenger(bank, tokenFactory)

	// Initialize the decorator for the custom messenger
	messengerDecoratorOpt := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(bank, tokenFactoryMessenger),
	)

	// Register custom message handlers
	return []wasmkeeper.Option{
		queryPluginOpt,
		messengerDecoratorOpt,
	}
}
