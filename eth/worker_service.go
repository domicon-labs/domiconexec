package eth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	baseModel "github.com/ethereum/go-ethereum/eth/model"
	"github.com/ethereum/go-ethereum/eth/tool"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	confirmationBlockNum = 6 //6区块确认
)

var SatoshiToBitcoin = float64(100000000)

type WorkerService struct {
	gdb    *gorm.DB
	btcCli *rpcclient.Client
}

func NewWorkerService(
	gdb *gorm.DB,
	btcCli *rpcclient.Client,
) *WorkerService {
	return &WorkerService{
		gdb:    gdb,
		btcCli: btcCli,
	}
}

//func (ws *WorkerService) SetBlockHeight(ctx context.Context, req reqs.SetBlockHeightReq) error {
//	err := ws.UpdateChain(ctx, req.ChainMagicNumber, int64(req.BlockHeight))
//	if err != nil {
//		return err
//	}
//
//	return nil
//}

//func (ws *WorkerService) SyncBlock(ctx context.Context, req reqs.SyncBlockReq) error {
//	syncBlockNumber := req.BlockHeight
//	chainId := req.ChainMagicNumber
//
//	beginBlockNumber := syncBlockNumber
//	endBlockNumber := syncBlockNumber
//
//	// 遍历获取block
//	blockNumberAndBlockMap, blockNumberAndBlockHeaderMap, err := ws.GetBlocks(ctx, beginBlockNumber, endBlockNumber)
//	if err != nil {
//		return err
//	}
//
//	//保存区块
//	err = ws.SaveBlocks(ctx, chainId, blockNumberAndBlockHeaderMap, blockNumberAndBlockMap)
//	if err != nil {
//		return err
//	}
//
//	// 保存交易
//	err = ws.SaveTransactions(ctx, chainId, blockNumberAndBlockMap)
//	if err != nil {
//		return err
//	}
//
//	// 保存文件
//	err = ws.SaveFiles(ctx, chainId, blockNumberAndBlockMap)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}

//func (ws *WorkerService) ShowTransaction(ctx context.Context, req reqs.ShowTransactionReq) (*resp.ShowTransactionData, error) {
//	q := baseQuery.Use(ws.db.GDB())
//
//	transaction, err := q.BaseTransaction.WithContext(ctx).
//		Where(q.BaseTransaction.ChainMagicNumber.Eq(req.ChainMagicNumber), q.BaseTransaction.TransactionHash.Eq(req.TransactionHash)).
//		First()
//	if err != nil {
//		return nil, err
//	}
//
//	isSubmit := domain.TransactionType(uint(transaction.TransactionType)) == domain.SubmitTransactionType
//
//	var fileData string
//	if isSubmit {
//		file, err := q.BaseFile.WithContext(ctx).
//			Where(q.BaseFile.ChainMagicNumber.Eq(req.ChainMagicNumber), q.BaseFile.SourceHash.Eq(transaction.SourceHash)).
//			First()
//		if err != nil {
//			return nil, err
//		}
//		fileData = file.Data
//	}
//
//	data := &resp.ShowTransactionData{
//		ChainID:           transaction.ChainID,
//		TransactionHash:   transaction.TransactionHash,
//		BlockNumber:       transaction.BlockNumber,
//		BlockHash:         transaction.BlockHash,
//		BlockTimestamp:    transaction.BlockTimestamp,
//		TransactionIndex:  transaction.TransactionIndex,
//		TransactionType:   transaction.TransactionType,
//		FromAddress:       transaction.FromAddress,
//		ToAddress:         transaction.ToAddress,
//		TransactionStatus: transaction.TransactionStatus,
//		InputData:         transaction.InputData,
//		TransactionValue:  transaction.TransactionValue,
//		Nonce:             transaction.Nonce,
//		GasPrice:          transaction.GasPrice.String(),
//		GasUsed:           transaction.GasUsed.String(),
//		IsSubmit:          isSubmit,
//		SourceHash:        transaction.SourceHash,
//		FileData:          fileData,
//	}
//
//	return data, nil
//}

//func (ws *WorkerService) Run(ctx context.Context, chainMagicNumber string, chainName string) error {
//	var err error
//
//	// 获取当前区块高度
//	currentBlockNumber, err := ws.GetCurrentBlockNumber(ctx)
//	if err != nil {
//		return err
//	}
//	customlogger.InfoZ(fmt.Sprintf("current block number: %d", currentBlockNumber))
//
//	// 读取数据库中的区块高度
//	presentBlockNumber, err := ws.GetPresentBlockNumber(ctx, chainMagicNumber, chainName)
//	if err != nil {
//		return err
//	}
//	customlogger.Infof("present block number: %v", presentBlockNumber)
//
//	// 如果当前区块高度等于数据库中的区块高度，则不处理
//	if presentBlockNumber >= currentBlockNumber {
//		customlogger.Infof("当前链上区块高度等于数据库中已同步的区块高度")
//		return nil
//	}
//
//	beginBlockNumber := presentBlockNumber + 1
//	endBlockNumber := los.Min([]int64{presentBlockNumber + 6, currentBlockNumber})
//
//	// 遍历获取block
//	blockNumberAndBlockMap, blockNumberAndBlockHeaderMap, err := ws.GetBlocks(ctx, beginBlockNumber, endBlockNumber)
//	if err != nil {
//		return err
//	}
//
//	//保存区块
//	err = ws.SaveBlocks(ctx, chainMagicNumber, blockNumberAndBlockHeaderMap, blockNumberAndBlockMap)
//	if err != nil {
//		return err
//	}
//
//	// 保存交易
//	err = ws.SaveTransactions(ctx, chainMagicNumber, blockNumberAndBlockMap)
//	if err != nil {
//		return err
//	}
//
//	// 保存文件
//	err = ws.SaveFiles(ctx, chainMagicNumber, blockNumberAndBlockMap)
//	if err != nil {
//		return err
//	}
//
//	// 更新当前区块高度
//	err = ws.UpdateChain(ctx, chainMagicNumber, endBlockNumber)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
//

// 获取链上当前区块高度
func (ws *WorkerService) GetCurrentBlockNumber(ctx context.Context) (int64, error) {
	// 获取最新区块哈希
	blockCount, err := ws.btcCli.GetBlockCount()
	if err != nil {
		log.Error("Error getting block count", "err", err)
		return 0, err
	}

	log.Info("Block Count", "blockCount", blockCount)

	return blockCount, nil
}

func (ws *WorkerService) GetPresentBlockNumber(ctx context.Context, chainMagicNumber string, chainName string) (int64, error) {
	var gormdb *gorm.DB
	var bc baseModel.BaseChain

	now := tool.TimeStampNowSecond()

	gormdb = ws.gdb.WithContext(ctx).
		Where(baseModel.BaseChain{ChainMagicNumber: chainMagicNumber}).
		Attrs(baseModel.BaseChain{ChainName: chainName, CurrentHeight: 0, CreateAt: now}).
		FirstOrCreate(&bc)
	if gormdb.Error != nil {
		return 0, gormdb.Error
	}
	return int64(bc.CurrentHeight), nil
}

// 更新当前区块高度
func (ws *WorkerService) UpdateChain(ctx context.Context, chainMagicNumber string, blockHeight int64) error {
	var gormdb *gorm.DB
	var bc baseModel.BaseChain

	gormdb = ws.gdb.WithContext(ctx).
		Where(baseModel.BaseChain{ChainMagicNumber: chainMagicNumber}).
		First(&bc)
	if gormdb.Error != nil {
		return gormdb.Error
	}

	now := tool.TimeStampNowSecond()

	bc.CurrentHeight = uint64(blockHeight)
	bc.CreateAt = now

	gormdb = ws.gdb.WithContext(ctx).Save(&bc)
	if gormdb.Error != nil {
		return gormdb.Error
	}

	return nil
}

func (ws *WorkerService) GetBlocks(ctx context.Context, from int64, to int64) (map[int64]*btcjson.GetBlockVerboseResult, map[int64]*wire.BlockHeader, error) {
	blockNumberAndBlockVerboseMap := make(map[int64]*btcjson.GetBlockVerboseResult)
	blockNumberAndBlockHeaderMap := make(map[int64]*wire.BlockHeader)

	// 遍历获取block
	for i := from; i <= to; i++ {
		blockHeight := i

		// 根据区块高度获取区块哈希
		blockHash, err := ws.btcCli.GetBlockHash(blockHeight)
		if err != nil {
			log.Error("Error getting block hash by height", "blockHeight", blockHeight, "err", err)
			return nil, nil, errors.New("get block hash by height err:" + err.Error())
		}
		log.Info("get block hash by height", "blockHash", blockHash, "blockHeight", blockHeight)

		// 使用区块哈希获取区块详细信息
		blockVerbose, err := ws.btcCli.GetBlockVerbose(blockHash)
		if err != nil {
			log.Error("Error getting block verbose by hash", "blockHash", blockHash, "err", err)
			return nil, nil, errors.New("get block verbose by hash err:" + err.Error())
		}

		// 打印区块详细信息
		log.Info("get block verbose by hash", "blockHeight", blockHeight, "blockHash", blockHash, "blockTime", blockVerbose.Time, "numberOfTransactions", len(blockVerbose.Tx))

		blockNumberAndBlockVerboseMap[i] = blockVerbose

		// 使用最新区块哈希获取区块详细信息
		blockHeader, err := ws.btcCli.GetBlockHeader(blockHash)
		if err != nil {
			log.Error("Error getting block header by hash", "blockHash", blockHash, "err", err)
			return nil, nil, errors.New("get block header by hash err:" + err.Error())
		}

		// 打印区块详细信息
		log.Info("get block header by hash", "blockHeight", blockHeight, "blockHash", blockHash, "blockTimestamp", blockHeader.Timestamp)

		blockNumberAndBlockHeaderMap[i] = blockHeader
	}

	return blockNumberAndBlockVerboseMap, blockNumberAndBlockHeaderMap, nil
}

// 保存区块
func (ws *WorkerService) SaveBlocks(ctx context.Context, chainMagicNumber string, blockNumberAndBlockHeaderMap map[int64]*wire.BlockHeader, blockNumberAndBlockVerboseMap map[int64]*btcjson.GetBlockVerboseResult) error {
	// 遍历获取block
	blockModels := make([]baseModel.BaseBlock, 0)

	now := tool.TimeStampNowSecond()

	for blockNumber, _ := range blockNumberAndBlockHeaderMap {
		block := blockNumberAndBlockVerboseMap[blockNumber]
		blockModels = append(blockModels, baseModel.BaseBlock{
			ChainMagicNumber: chainMagicNumber,
			BlockHeight:      block.Height,
			BlockHash:        block.Hash,
			Confirmations:    block.Confirmations,
			StrippedSize:     block.StrippedSize,
			Size:             block.Size,
			Weight:           block.Weight,
			MerkleRoot:       block.MerkleRoot,
			TransactionCnt:   uint32(len(block.Tx)),
			BlockTime:        block.Time,
			Nonce:            block.Nonce,
			Bits:             block.Bits,
			Difficulty:       block.Difficulty,
			PreviousHash:     block.PreviousHash,
			NextHash:         block.NextHash,
			CreateAt:         now,
		})
	}

	var gormdb *gorm.DB

	// 保存区块
	gormdb = ws.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&blockModels)

	if gormdb.Error != nil {
		return gormdb.Error
	}

	return nil
}

// 保存交易
func (ws *WorkerService) SaveTransactions(ctx context.Context, chainMagicNumber string, blockNumberAndBlockVerboseMap map[int64]*btcjson.GetBlockVerboseResult) error {
	hashes := make([]string, 0)

	for _, block := range blockNumberAndBlockVerboseMap {
		for _, tx := range block.Tx {
			hashes = append(hashes, tx)
		}
	}

	transactionModels := make([]baseModel.BaseTransaction, 0)

	now := tool.TimeStampNowSecond()

	// 校验交易内容
	for _, block := range blockNumberAndBlockVerboseMap {
		for _, tx := range block.Tx {
			txid, err := chainhash.NewHashFromStr(tx)
			if err != nil {
				log.Error("Error converting txid string to hash", "tx", tx, "err", err)
				continue
			}

			transactionVerbose, err := ws.btcCli.GetRawTransactionVerbose(txid)
			if err != nil {
				log.Error("Error getting transaction by hash", "txid", txid, "err", err)
				continue
			}

			log.Info("Transaction Timestamp", "transactionTime", transactionVerbose.Time)

			vinDataBytes, err := json.Marshal(transactionVerbose.Vin)
			if err != nil {
				log.Error("Error marshaling vin data", "err", err)
				continue
			}

			voutDataBytes, err := json.Marshal(transactionVerbose.Vout)
			if err != nil {
				log.Error("Error marshaling vout data", "err", err)
				continue
			}

			transactionModels = append(transactionModels, baseModel.BaseTransaction{
				ChainMagicNumber: chainMagicNumber,
				Hex:              transactionVerbose.Hex,
				Txid:             transactionVerbose.Txid,
				TransactionHash:  transactionVerbose.Hash,
				Size:             transactionVerbose.Size,
				Vsize:            transactionVerbose.Vsize,
				Weight:           transactionVerbose.Weight,
				LockTime:         transactionVerbose.LockTime,
				Vin:              vinDataBytes,
				Vout:             voutDataBytes,
				BlockHash:        transactionVerbose.BlockHash,
				Confirmations:    transactionVerbose.Confirmations,
				TransactionTime:  transactionVerbose.Time,
				BlockTime:        transactionVerbose.Blocktime,
				CreateAt:         now,
			})
		}
	}

	log.Info("number of transactions", "number", len(transactionModels))

	var gormdb *gorm.DB

	// 保存交易
	gormdb = ws.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&transactionModels)

	if gormdb.Error != nil {
		return gormdb.Error
	}

	return nil
}

//// 保存文件
//func (ws *WorkerService) SaveFiles(ctx context.Context, chainMagicNumber string, blockNumberAndBlockMap map[int64]*btcjson.GetBlockVerboseResult) error {
//	q := baseQuery.Use(ws.db.GDB())
//
//	fileModels := make([]*baseModel.BaseFile, 0)
//
//	//now := tool.TimeStampNowSecond()
//
//	//for _, block := range blockNumberAndBlockMap {
//	//		for _, tx := range block.Transactions {
//	//			if tx.Type != uint(domain.SubmitTransactionType) {
//	//				continue
//	//			}
//	//			file, err := ws.eth.FileByHash(ctx, tx.SourceHash)
//	//			if err != nil {
//	//				customlogger.ErrorZ(err.Error())
//	//				continue
//	//			}
//	//
//	//			fileModels = append(fileModels, &baseModel.BaseFile{
//	//				ChainID:         chainId,
//	//				SourceHash:      tx.SourceHash.Hex(),
//	//				Sender:          file.Sender.Hex(),
//	//				Submitter:       file.Submitter.Hex(),
//	//				Length:          uint64(file.Length),
//	//				Index:           uint64(file.Index),
//	//				Commitment:      utils.ByteToHex(file.Commitment),
//	//				Data:            utils.ByteToHex(file.Data),
//	//				Sign:            utils.ByteToHex(file.Sign),
//	//				TransactionHash: tx.Hash.Hex(),
//	//				CreateAt:        now,
//	//			})
//	//		}
//	//}
//
//	customlogger.Infof("文件数量: %v", len(fileModels))
//	// 保存文件
//	err := q.BaseFile.WithContext(ctx).Clauses(clause.Insert{Modifier: "IGNORE"}).CreateInBatches(fileModels, 30)
//
//	if err != nil {
//		return err
//	}
//	return nil
//}
