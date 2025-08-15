package watchers

import (
	"strings"

	"github.com/switchlyprotocol/switchlynode/v3/config"
)

////////////////////////////////////////////////////////////////////////////////////////
// Init
////////////////////////////////////////////////////////////////////////////////////////

var switchlynodeURL string

func init() {
	config.Init()
	switchlynodeURL = config.GetBifrost().Switchly.ChainHost
	if !strings.HasPrefix(switchlynodeURL, "http") {
		switchlynodeURL = "http://" + switchlynodeURL
	}
}
