package thorchain

import (
	cosmos "github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

// DummyEventMgr used for test purpose , and it implement EventManager interface
type DummyEventMgr struct{}

func NewDummyEventMgr() *DummyEventMgr {
	return &DummyEventMgr{}
}

func (m *DummyEventMgr) EmitEvent(ctx cosmos.Context, evt EmitEventItem) error      { return nil }
func (m *DummyEventMgr) EmitGasEvent(ctx cosmos.Context, gasEvent *EventGas) error  { return nil }
func (m *DummyEventMgr) EmitSwapEvent(ctx cosmos.Context, swap *EventSwap) error    { return nil }
func (m *DummyEventMgr) EmitFeeEvent(ctx cosmos.Context, feeEvent *EventFee) error  { return nil }
func (m *DummyEventMgr) EmitPoolEvent(ctx cosmos.Context, poolEvt *EventPool) error { return nil }
