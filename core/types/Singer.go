package types

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	kzg "github.com/domicon-labs/kzg-sdk"
	"github.com/ethereum/go-ethereum/crypto"
	//"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/params"
	"os"
	"strings"
)
const dSrsSize = 1 << 16
type SingerTool struct {
	config  *params.ChainConfig
	prv *ecdsa.PrivateKey
}

func NewSingerTool(conf *params.ChainConfig,prv *ecdsa.PrivateKey) *SingerTool {
	return &SingerTool{
		config: conf,
		prv: prv,
	}
}

func (s *SingerTool) Sign(da *DA) ([]byte,error) {
	singer := NewEIP155FdSigner(s.config.ChainID)
	h := singer.Hash(da)
	return crypto.Sign(h.Bytes(),s.prv)
}

func (s *SingerTool) Verify(da *DA) (bool,error) {
	if da.Length != uint64(len(da.Data)) {
		return false,errors.New("da data length is not match")
	}
	currentPath, _ := os.Getwd()
	path := strings.Split(currentPath,"/build")[0] + "/srs"
	domiconSDK,err := kzg.InitDomiconSdk(dSrsSize,path)
	if err != nil {
		return false,err
	}

	digst,err := domiconSDK.GenerateDataCommit(da.Data)
	if err != nil {
		return false,errors.New("GenerateDataCommit failed")
	}
	x := digst.X.Marshal()
	y := digst.Y.Marshal()

	if (bytes.Compare(x,da.Commitment.X.Marshal()) == 0) && (bytes.Compare(y,da.Commitment.Y.Marshal()) == 0){
		return true,nil
	}
	return false,errors.New("commit is not match with da")
}