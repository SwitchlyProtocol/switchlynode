package types

const (
	// ModuleName name of Switchly module
	ModuleName = "switchly"
	// DefaultCodespace is the same as ModuleName
	DefaultCodespace = ModuleName
	// ReserveName the module account name to keep reserve
	ReserveName = "reserve"
	// AsgardName the module account name to keep asgard fund
	AsgardName = "asgard"
	// BondName the name of account used to store bond
	BondName = "bond"
	// LendingName
	LendingName = "lending"
	// AffiliateCollectorName the name of the account used to store swtc for affiliate fee swaps
	AffiliateCollectorName = "affiliate_collector"
	// TreasuryName the name of the account used for treasury governance
	TreasuryName = "treasury"
	// SWTCPoolName the name of the account used to track SWTCPool
	SWTCPoolName = "swtc_pool"
	// TCYClaimingName the name of the account used to track claming funds from $TCY
	TCYClaimingName = "tcy_claim"
	// TCYStakeName the name of the account used to track stake funds from $TCY
	TCYStakeName = "tcy_stake"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	RouterKey = ModuleName

	QuerierRoute = ModuleName
)
