package db

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gorm.io/gorm"
)

// 创建日志表格模型
type Log struct {
	gorm.Model
	TxHash     string `gorm:"primaryKey"`
	LogIndex   int
	Address    string `gorm:"not null"`
	BlockNum   int64
	BlockHash  string `gorm:"not null"`
	Removed    bool
	Data       string
	Topic0     string `gorm:"not null"`
	Topic1     string
	Topic2     string
	Topic3     string
}

func AddLog(tx *gorm.DB, log Log) error {
	res := tx.Create(&log)
	if res.Error != nil {
		tx.Rollback()
		return res.Error
	}
	return nil
}

func AddBatchLogs(tx *gorm.DB, logs []*types.Log) error {
	// 遍历每个区块，依次插入数据库
	for _, logIns := range logs {
		tpcLength := len(logIns.Topics)
		logWrite := Log{
			TxHash: logIns.TxHash.Hex(),
			LogIndex: int(logIns.Index),
			Address: logIns.Address.Hex(),
			BlockNum: int64(logIns.BlockNumber),
			BlockHash: logIns.BlockHash.Hex(),
			Removed: logIns.Removed,
			Data: common.Bytes2Hex(logIns.Data),
		}

		switch tpcLength {
		case 0:
			logWrite.Topic0 = logIns.Topics[0].Hex()
		case 1:
			logWrite.Topic1 = logIns.Topics[0].Hex()
		case 2:
			logWrite.Topic2 = logIns.Topics[0].Hex()
		case 3:
			logWrite.Topic3 = logIns.Topics[0].Hex()
		}

		result := tx.Create(&logWrite)
		if result.Error != nil {
			// 插入失败，回滚事务并返回错误
			tx.Rollback()
			return result.Error
		}
	}
	return nil
}

func GetLogByHash(db *gorm.DB,blockHash common.Hash) ([]*types.Log,error) {
	var log Log
	tx := db.First(&log, "block_hash = ?", blockHash.Hex())
	if tx.Error == nil {

	}
	errstr := fmt.Sprintf("can not find block with given blockHash :%s",blockHash.Hex())
	return nil,errors.New(errstr)
}

func DeleteLogWithTxHash(db *gorm.DB,txHash common.Hash) error {
	var log Log
	err := db.Where("tx_hash = ? ", txHash.Hex()).Delete(&log).Error
	if err != nil {
		db.Rollback()
		return err
	}
	return nil
}
