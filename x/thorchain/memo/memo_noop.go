package thorchain

import (
	"fmt"
)

type NoOpMemo struct {
	MemoBase
	Action string
}

// String implement fmt.Stringer
func (m NoOpMemo) String() string {
	if len(m.Action) == 0 {
		return "noop"
	}
	return fmt.Sprintf("noop:%s", m.Action)
}

// NewNoOpMemo create a new instance of NoOpMemo
func NewNoOpMemo(action string) NoOpMemo {
	return NoOpMemo{
		MemoBase: MemoBase{TxType: TxNoOp},
		Action:   action,
	}
}

// ParseNoOpMemo try to parse the memo
func (p *parser) ParseNoOpMemo() (NoOpMemo, error) {
	return NewNoOpMemo(p.get(1)), p.Error()
}
