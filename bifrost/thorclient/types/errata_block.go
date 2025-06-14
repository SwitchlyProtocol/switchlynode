package types

import "github.com/switchlyprotocol/switchlynode/v1/common"

type ErrataBlock struct {
	Height int64
	Txs    []ErrataTx
}

type ErrataTx struct {
	TxID  common.TxID
	Chain common.Chain
}
