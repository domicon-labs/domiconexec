// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package ethapi implements the general Ethereum API functions.
package ethapi

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool/filedatapool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	// General Ethereum API
	//SyncProgress() ethereum.SyncProgress

	//SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	//FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, error)
	ChainDb() ethdb.Database
	AccountManager() *accounts.Manager
	ExtRPCEnabled() bool
	RPCGasCap() uint64            // global gas cap for eth_call over rpc: DoS protection
	//RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	RPCTxFeeCap() float64         // global tx fee cap for all transaction related APIs
	UnprotectedAllowed() bool     // allows only for EIP155 transactions.

	// Blockchain API
	SetHead(number uint64)
	//HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	//HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	//HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error)
	//CurrentHeader() *types.Header
	CurrentBlock() *types.Header
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error)
	GetTd(ctx context.Context) *big.Int

	// FileData pool API
	UploadFileDataByParams(sender, submitter common.Address, index, length, gasPrice uint64, commitment, data, signData []byte, txHash common.Hash) error
	UploadFileData(data []byte) error
	CheckSelfState(blockNr rpc.BlockNumber) (string,error)
	GetFileDataByHash(hash common.Hash) (*types.DA,filedatapool.DISK_FILEDATA_STATE,error)
	GetFileDataByCommitment(comimt []byte) (*types.DA, error)
	DiskSaveFileDataWithHash(hash common.Hash) (bool, error)
	DiskSaveFileDatas(hashes []common.Hash,blockNrOrHash rpc.BlockNumberOrHash) (bool, error)
	BatchSaveFileDataWithHashes(hashes rpc.TxHashes) ([]bool, []error)
	ChangeCurrentState(state int, number rpc.BlockNumber) bool
	SubscribeNewFileDataEvent(chan<- core.NewFileDataEvent) event.Subscription
	
	ChainConfig() *params.ChainConfig

	HistoricalRPCService() *rpc.Client
	Genesis() *types.Block
}

func GetAPIs(apiBackend Backend) []rpc.API {
	nonceLock := new(AddrLocker)
	return []rpc.API{
		 {
			Namespace: "eth",
			Service:   NewBlockChainAPI(apiBackend),
		},  {
			Namespace: "eth",
			Service:   NewFileDataAPI(apiBackend),
		}, {
			Namespace: "debug",
			Service:   NewDebugAPI(apiBackend),
		}, {
			Namespace: "eth",
			Service:   NewEthereumAccountAPI(apiBackend.AccountManager()),
		}, {
			Namespace: "personal",
			Service:   NewPersonalAccountAPI(apiBackend, nonceLock),
		},
	}
}