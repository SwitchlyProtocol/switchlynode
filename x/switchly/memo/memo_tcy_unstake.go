package switchly

import (
	"cosmossdk.io/math"
)

type SWCYUnstakeMemo struct {
	MemoBase
	BasisPoints math.Uint
}

func NewSWCYUnstakeMemo(basisPoints math.Uint) SWCYUnstakeMemo {
	return SWCYUnstakeMemo{
		MemoBase:    MemoBase{TxType: TxSWCYUnstake},
		BasisPoints: basisPoints,
	}
}

func (p *parser) ParseSWCYUnstakeMemo() (SWCYUnstakeMemo, error) {
	bps := p.getUint(1, true, 0)
	return NewSWCYUnstakeMemo(bps), p.Error()
}
