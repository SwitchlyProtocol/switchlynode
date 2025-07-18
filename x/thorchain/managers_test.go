package thorchain

import (
	"errors"

	"github.com/blang/semver"
	. "gopkg.in/check.v1"
)

type ManagersTestSuite struct{}

var _ = Suite(&ManagersTestSuite{})

func (ManagersTestSuite) TestManagers(c *C) {
	_, mgr := setupManagerForTest(c)
	ver := semver.MustParse("0.0.1")

	gasMgr, err := GetGasManager(ver, mgr.Keeper())
	c.Assert(gasMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	eventMgr, err := GetEventManager(ver)
	c.Assert(eventMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	// Versioning has been removed from GetTxOutStore.
	txOutStore, err := GetTxOutStore(ver, mgr.Keeper(), mgr.EventMgr(), gasMgr)
	c.Assert(txOutStore, NotNil)
	c.Assert(err, IsNil)

	vaultMgr, err := GetNetworkManager(ver, mgr.Keeper(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(vaultMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	// Versioning has been removed from GetValidatorManager.
	validatorManager, err := GetValidatorManager(ver, mgr.Keeper(), mgr.NetworkMgr(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(validatorManager, NotNil)
	c.Assert(err, IsNil)

	observerMgr, err := GetObserverManager(ver)
	c.Assert(observerMgr, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	swapQueue, err := GetSwapQueue(ver, mgr.Keeper())
	c.Assert(swapQueue, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)

	// Versioning has been removed from GetSwapper.
	swapper, err := GetSwapper(ver)
	c.Assert(swapper, NotNil)
	c.Assert(err, IsNil)

	slasher, err := GetSlasher(ver, mgr.Keeper(), mgr.EventMgr())
	c.Assert(slasher, IsNil)
	c.Assert(err, NotNil)
	c.Assert(errors.Is(err, errInvalidVersion), Equals, true)
}
