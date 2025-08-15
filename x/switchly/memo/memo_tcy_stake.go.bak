package switchly

type SWCYStakeMemo struct {
	MemoBase
}

func NewSWCYStakeMemo() SWCYStakeMemo {
	return SWCYStakeMemo{
		MemoBase: MemoBase{TxType: TxSWCYStake},
	}
}

func (p *parser) ParseSWCYStakeMemo() (SWCYStakeMemo, error) {
	return NewSWCYStakeMemo(), p.Error()
}
