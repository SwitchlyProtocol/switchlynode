package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/gogo/protobuf/jsonpb"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// EmptyAsset empty asset, not valid
	EmptyAsset = Asset{Chain: EmptyChain, Symbol: "", Ticker: "", Synth: false}
	// ATOMAsset ATOM
	ATOMAsset = Asset{Chain: GAIAChain, Symbol: "ATOM", Ticker: "ATOM", Synth: false}
	// BNBBEP20Asset BNB
	BNBBEP20Asset = Asset{Chain: BSCChain, Symbol: "BNB", Ticker: "BNB", Synth: false}
	// BTCAsset BTC
	BTCAsset = Asset{Chain: BTCChain, Symbol: "BTC", Ticker: "BTC", Synth: false}
	// LTCAsset BTC
	LTCAsset = Asset{Chain: LTCChain, Symbol: "LTC", Ticker: "LTC", Synth: false}
	// BCHAsset BCH
	BCHAsset = Asset{Chain: BCHChain, Symbol: "BCH", Ticker: "BCH", Synth: false}
	// DOGEAsset DOGE
	DOGEAsset = Asset{Chain: DOGEChain, Symbol: "DOGE", Ticker: "DOGE", Synth: false}
	// ETHAsset ETH
	ETHAsset = Asset{Chain: ETHChain, Symbol: "ETH", Ticker: "ETH", Synth: false}
	// BaseAsset ETH
	BaseETHAsset = Asset{Chain: BASEChain, Symbol: "ETH", Ticker: "ETH", Synth: false}
	// AVAXAsset AVAX
	AVAXAsset = Asset{Chain: AVAXChain, Symbol: "AVAX", Ticker: "AVAX", Synth: false}
	// XRPAsset XRP
	XRPAsset = Asset{Chain: XRPChain, Symbol: "XRP", Ticker: "XRP", Synth: false}
	// XLMAsset XLM
	XLMAsset = Asset{Chain: StellarChain, Symbol: "XLM", Ticker: "XLM", Synth: false}
	// XLMUSDC USDC on Stellar
	XLMUSDC = Asset{Chain: StellarChain, Symbol: "USDC", Ticker: "USDC", Synth: false}
	// SwitchNative SWITCH on switchly
	SwitchNative = Asset{Chain: SWITCHLYChain, Symbol: "SWITCH", Ticker: "SWITCH", Synth: false}
	RUJI         = Asset{Chain: SWITCHLYChain, Symbol: "RUJI", Ticker: "RUJI", Synth: false}
	TCY          = Asset{Chain: SWITCHLYChain, Symbol: "TCY", Ticker: "TCY", Synth: false}
	TOR          = Asset{Chain: SWITCHLYChain, Symbol: "TOR", Ticker: "TOR", Synth: false}
	SWITCHLYBTC  = Asset{Chain: SWITCHLYChain, Symbol: "BTC", Ticker: "BTC", Synth: false}
)

var _ sdk.CustomProtobufType = (*Asset)(nil)

// NewAsset parse the given input into Asset object
func NewAsset(input string) (Asset, error) {
	var err error
	var asset Asset
	var sym string
	var parts []string
	re := regexp.MustCompile("[~./-]")

	match := re.FindString(input)

	switch match {
	case "~":
		parts = strings.SplitN(input, match, 2)
		asset.Trade = true
	case "/":
		parts = strings.SplitN(input, match, 2)
		asset.Synth = true
	case "-":
		parts = strings.SplitN(input, match, 2)
		asset.Secured = true
	case ".":
		parts = strings.SplitN(input, match, 2)
	case "":
		parts = []string{input}
	}
	if len(parts) == 1 {
		asset.Chain = SWITCHLYChain
		sym = parts[0]
	} else {
		asset.Chain, err = NewChain(parts[0])
		if err != nil {
			return EmptyAsset, err
		}
		sym = parts[1]
	}

	asset.Symbol, err = NewSymbol(sym)
	if err != nil {
		return EmptyAsset, err
	}

	parts = strings.SplitN(sym, "-", 2)
	asset.Ticker, err = NewTicker(parts[0])
	if err != nil {
		return EmptyAsset, err
	}

	return asset, nil
}

func NewAssetWithShortCodes(version semver.Version, input string) (Asset, error) {
	return NewAssetWithShortCodesV3_1_0(input)
}

func NewAssetWithShortCodesV3_1_0(input string) (Asset, error) {
	shorts := make(map[string]string)

	shorts[ATOMAsset.ShortCode()] = ATOMAsset.String()
	shorts[AVAXAsset.ShortCode()] = AVAXAsset.String()
	shorts[BCHAsset.ShortCode()] = BCHAsset.String()
	shorts[BNBBEP20Asset.ShortCode()] = BNBBEP20Asset.String()
	shorts[BTCAsset.ShortCode()] = BTCAsset.String()
	shorts[DOGEAsset.ShortCode()] = DOGEAsset.String()
	shorts[ETHAsset.ShortCode()] = ETHAsset.String()
	shorts[LTCAsset.ShortCode()] = LTCAsset.String()
	shorts[SwitchNative.ShortCode()] = SwitchNative.String()
	shorts[BaseETHAsset.ShortCode()] = BaseETHAsset.String()
	shorts[XRPAsset.ShortCode()] = XRPAsset.String()
	shorts[XLMAsset.ShortCode()] = XLMAsset.String()
	shorts[XLMUSDC.ShortCode()] = XLMUSDC.String()

	long, ok := shorts[input]
	if ok {
		input = long
	}

	return NewAsset(input)
}

func (a Asset) Valid() error {
	if err := a.Chain.Valid(); err != nil {
		return fmt.Errorf("invalid chain: %w", err)
	}
	if err := a.Symbol.Valid(); err != nil {
		return fmt.Errorf("invalid symbol: %w", err)
	}
	if (a.Synth && a.Trade) || (a.Trade && a.Secured) || (a.Secured && a.Synth) {
		return fmt.Errorf("assets can only be one of trade, synth or secured")
	}
	if a.Synth && a.Chain.IsSWITCHLYChain() {
		return fmt.Errorf("synth asset cannot have chain SWITCHLY: %s", a)
	}
	if a.Trade && a.Chain.IsSWITCHLYChain() {
		return fmt.Errorf("trade asset cannot have chain SWITCHLY: %s", a)
	}
	if a.Secured && a.Chain.IsSWITCHLYChain() {
		return fmt.Errorf("secured asset cannot have chain SWITCHLY: %s", a)
	}
	return nil
}

// Equals determinate whether two assets are equivalent
func (a Asset) Equals(a2 Asset) bool {
	return a.Chain.Equals(a2.Chain) && a.Symbol.Equals(a2.Symbol) && a.Ticker.Equals(a2.Ticker) && a.Synth == a2.Synth && a.Trade == a2.Trade && a.Secured == a2.Secured
}

func (a Asset) GetChain() Chain {
	if a.Synth || a.Trade || a.Secured {
		return SWITCHLYChain
	}
	return a.Chain
}

// Get layer1 asset version
func (a Asset) GetLayer1Asset() Asset {
	if !a.IsSyntheticAsset() && !a.IsTradeAsset() && !a.IsSecuredAsset() {
		return a
	}
	return Asset{
		Chain:   a.Chain,
		Symbol:  a.Symbol,
		Ticker:  a.Ticker,
		Synth:   false,
		Trade:   false,
		Secured: false,
	}
}

// Get synthetic asset of asset
func (a Asset) GetSyntheticAsset() Asset {
	if a.IsSyntheticAsset() {
		return a
	}
	return Asset{
		Chain:  a.Chain,
		Symbol: a.Symbol,
		Ticker: a.Ticker,
		Synth:  true,
	}
}

// Get trade asset of asset
func (a Asset) GetTradeAsset() Asset {
	if a.IsTradeAsset() {
		return a
	}
	return Asset{
		Chain:  a.Chain,
		Symbol: a.Symbol,
		Ticker: a.Ticker,
		Trade:  true,
	}
}

// Get secured asset of asset
func (a Asset) GetSecuredAsset() Asset {
	if a.IsSecuredAsset() {
		return a
	}
	return Asset{
		Chain:   a.Chain,
		Symbol:  a.Symbol,
		Ticker:  a.Ticker,
		Secured: true,
	}
}

// Get derived asset of asset
func (a Asset) GetDerivedAsset() Asset {
	return Asset{
		Chain:  SWITCHLYChain,
		Symbol: a.Symbol,
		Ticker: a.Ticker,
		Synth:  false,
	}
}

// Check if asset is a pegged asset
func (a Asset) IsSyntheticAsset() bool {
	return a.Synth
}

func (a Asset) IsTradeAsset() bool {
	return a.Trade
}

func (a Asset) IsSecuredAsset() bool {
	return a.Secured
}

func (a Asset) IsVaultAsset() bool {
	return a.IsSyntheticAsset()
}

// Check if asset is a derived asset
func (a Asset) IsDerivedAsset() bool {
	return !a.Synth && !a.Trade && !a.Secured && a.GetChain().IsSWITCHLYChain() && !a.IsRune() && !a.IsTCY() && !a.IsRUJI()
}

// Native return native asset, only relevant on SWITCHLYChain
func (a Asset) Native() string {
	switch {
	case a.IsRune():
		return "switch"
	case a.Equals(TOR):
		return "tor"
	case a.Equals(TCY):
		return "tcy"
	case a.IsRUJI():
		return "x/ruji"
	}

	return strings.ToLower(a.String())
}

// IsEmpty will be true when any of the field is empty, chain,symbol or ticker
func (a Asset) IsEmpty() bool {
	return a.Chain.IsEmpty() || a.Symbol.IsEmpty() || a.Ticker.IsEmpty()
}

// String implement fmt.Stringer , return the string representation of Asset
func (a Asset) String() string {
	div := "."
	if a.Synth {
		div = "/"
	}
	if a.Trade {
		div = "~"
	}
	if a.Secured {
		div = "-"
	}
	return fmt.Sprintf("%s%s%s", a.Chain.String(), div, a.Symbol.String())
}

// ShortCode returns the short code for the asset.
func (a Asset) ShortCode() string {
	switch a.String() {
	case "SWITCHLY.SWITCH":
		return "r"
	case "GAIA.ATOM":
		return "g"
	case "DOGE.DOGE":
		return "d"
	case "LTC.LTC":
		return "l"
	case "BCH.BCH":
		return "c"
	case "AVAX.AVAX":
		return "a"
	case "BTC.BTC":
		return "b"
	case "BSC.BNB":
		return "n"
	case "BASE.ETH":
		return "f"
	case "XRP.XRP":
		return "x"
	case "XLM.XLM":
		return "m"
	case "XLM.USDC":
		return "u"
	default:
		return ""
	}
}

// IsGasAsset check whether asset is base asset used to pay for gas
func (a Asset) IsGasAsset() bool {
	gasAsset := a.GetChain().GetGasAsset()
	if gasAsset.IsEmpty() {
		return false
	}
	return a.Equals(gasAsset)
}

// IsRune check whether asset is RUNE
func (a Asset) IsRune() bool {
	return a.Equals(SwitchNative)
}

// IsTCY is a helper function ,return true only when the asset represent TCY
func (a Asset) IsTCY() bool {
	return TCY.Equals(a)
}

func (a Asset) IsRUJI() bool {
	return RUJI.Equals(a)
}

// IsNative is a helper function, returns true when the asset is a native
// asset to SWITCHLYChain (ie switch, a synth, etc)
func (a Asset) IsNative() bool {
	return a.GetChain().IsSWITCHLYChain()
}

// MarshalJSON implement Marshaler interface
func (a Asset) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON implement Unmarshaler interface
func (a *Asset) UnmarshalJSON(data []byte) error {
	var err error
	var assetStr string
	if err = json.Unmarshal(data, &assetStr); err != nil {
		return err
	}
	if assetStr == "." {
		*a = EmptyAsset
		return nil
	}
	*a, err = NewAsset(assetStr)
	return err
}

// MarshalJSONPB implement jsonpb.Marshaler
func (a Asset) MarshalJSONPB(*jsonpb.Marshaler) ([]byte, error) {
	return a.MarshalJSON()
}

// UnmarshalJSONPB implement jsonpb.Unmarshaler
func (a *Asset) UnmarshalJSONPB(unmarshal *jsonpb.Unmarshaler, content []byte) error {
	return a.UnmarshalJSON(content)
}

// Replace pool name "." with a "-" for Mimir key checking.
func (a Asset) MimirString() string {
	return a.Chain.String() + "-" + a.Symbol.String()
}

func (a Asset) IsSwitch() bool {
	return a.Equals(SwitchNative)
}
