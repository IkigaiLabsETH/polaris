// SPDX-License-Identifier: BUSL-1.1
//
// Copyright (C) 2023, Berachain Foundation. All rights reserved.
// Use of this software is govered by the Business Source License included
// in the LICENSE file of this repository and at www.mariadb.com/bsl11.
//
// ANY USE OF THE LICENSED WORK IN VIOLATION OF THIS LICENSE WILL AUTOMATICALLY
// TERMINATE YOUR RIGHTS UNDER THIS LICENSE FOR THE CURRENT AND ALL OTHER
// VERSIONS OF THE LICENSED WORK.
//
// THIS LICENSE DOES NOT GRANT YOU ANY RIGHT IN ANY TRADEMARK OR LOGO OF
// LICENSOR OR ITS AFFILIATES (PROVIDED THAT YOU MAY USE A TRADEMARK OR LOGO OF
// LICENSOR AS EXPRESSLY REQUIRED BY THIS LICENSE).
//
// TO THE EXTENT PERMITTED BY APPLICABLE LAW, THE LICENSED WORK IS PROVIDED ON
// AN “AS IS” BASIS. LICENSOR HEREBY DISCLAIMS ALL WARRANTIES AND CONDITIONS,
// EXPRESS OR IMPLIED, INCLUDING (WITHOUT LIMITATION) WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, NON-INFRINGEMENT, AND
// TITLE.

package polaris

import (
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"

	"pkg.berachain.dev/polaris/cosmos/runtime/polaris/mempool"
	"pkg.berachain.dev/polaris/cosmos/runtime/polaris/miner"
	"pkg.berachain.dev/polaris/eth/core"
	"pkg.berachain.dev/polaris/eth/core/txpool"
	ethlog "pkg.berachain.dev/polaris/eth/log"
	"pkg.berachain.dev/polaris/eth/polar"
)

// TODO deprecate this

type EVMKeeper interface {
	SetPolaris(*polar.Polaris)
}

// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
//
//nolint:revive // readability.
type PolarisApp struct {
	// cosmos stuff
	*runtime.App
	logger    log.Logger
	clientCtx client.Context

	// ethereum stuff
	polaris   *polar.Polaris
	mempool   *mempool.WrappedGethTxPool
	hostChain core.PolarisHostChain
	Evmkeeper EVMKeeper
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (a *PolarisApp) RegisterAPIRoutes(apiSvr *api.Server, _ config.APIConfig) {
	// Pass the go-ethereum txpool to the handler, as well as the clientCtx so it can
	// broadcast transactions inserted into the mempool to comet.
	a.clientCtx = apiSvr.ClientCtx

	// TODO: TxPool initialization
	// Intializating the txpool here seems to be safe, but I think there may be a race condition here.
	// What we should really be doing is reading the ChainConfig directly from the database after `Load()` is called.
	// Opposed to getting it through the blockchain object, which won't be prepare'd until after BeginBlock(), I think
	// it is just happenstance that RegisterAPIRoutes is happen after the blockchain is prepared.
	//
	// Note: Once we are properly loading the state of the blockchain in `Load()` this issue should be formally resolved.
	txPool := txpool.NewTxPool(txpool.DefaultConfig, a.polaris.Blockchain().Config(), a.polaris.Blockchain())
	a.polaris.SetTxPool(txPool)

	// Now that we have the client context and the txpool, we can setup the mempool and miner.
	a.mempool.Setup(txPool, a.hostChain.GetConfigurationPlugin(), miner.NewTxSerializer(a.clientCtx))

	// We set the handler.
	a.polaris.SetHandler(miner.NewHandler(txPool, a.clientCtx))

	// Note: this is a bit of an awkward place to put this, but if you look in the Cosmos-SDK server:
	// https://github.com/cosmos/cosmos-sdk/blob/3db9528efb5fec1cccdb4e6f084c24ed195951b1/server/start.go#L504
	// You'll see that the API server is started right after `RegisterAPIRoutes` is called. So starting the
	// Polaris services here is quite oddly a semi logical place to do it (in lieu of having a custom
	// server even though it feels a little strange.
	// TODO: Really we should create a way for runtime modules to register services with the server package.
	// We suggest this to @tac0turtle.
	if err := a.polaris.StartServices(); err != nil {
		panic(err)
	}
}

// Load is called on application initialization and provides an opportunity to
// perform initialization logic. It returns an error if initialization fails.
// We shadow the Load function from cosmos-sdk/runtime/app.go in order to prime the blockchain
// and miner objects to allow the EVM to reach a consistent state before it begins processing blocks.
func (a *PolarisApp) Load(latest bool) error {
	if err := a.App.Load(latest); err != nil {
		return err
	}

	// Load EVM keeper or something?
	// TODO: PARSE POLARIS.TOML CORRECT AGAIN
	nodeCfg := polar.DefaultGethNodeConfig()
	// TODO: unfuck this
	nodeCfg.DataDir = "./.tmp/polaris"
	node, err := polar.NewGethNetworkingStack(nodeCfg)
	if err != nil {
		panic(err)
	}
	a.polaris = polar.New(
		polar.DefaultConfig(),
		a.hostChain,
		node,
		ethlog.FuncHandler(
			func(r *ethlog.Record) error {
				polarisGethLogger := a.logger.With("module", "polaris-geth")
				switch r.Lvl { //nolint:nolintlint,exhaustive // linter is bugged.
				case ethlog.LvlTrace, ethlog.LvlDebug:
					polarisGethLogger.Debug(r.Msg, r.Ctx...)
				case ethlog.LvlInfo, ethlog.LvlWarn:
					polarisGethLogger.Info(r.Msg, r.Ctx...)
				case ethlog.LvlError, ethlog.LvlCrit:
					polarisGethLogger.Error(r.Msg, r.Ctx...)
				}
				return nil
			}),
		nil,
	)
	// todo: this is horrid.
	a.Evmkeeper.SetPolaris(a.polaris)

	// Load the polaris runtime to warm the blockchain object.
	return nil
}