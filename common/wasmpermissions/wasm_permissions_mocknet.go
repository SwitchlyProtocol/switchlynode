//go:build mocknet
// +build mocknet

package wasmpermissions

var WasmPermissionsRaw = WasmPermissions{
	Permissions: map[string]WasmPermission{
		// Test code entry for TestQueryCodes
		"a8f1a38aa518864169e30ab482ea86558a817982a030b8888ea6dfa0cd700128": {
			Origin: "https://switchly.org",
			Deployers: map[string]bool{
				"tswitch1nrp4veflhpjnv87akxcpxln5lmc4z4kkdsdpd5": true,
				"tswitch1770gl4x5u7aauzlc7su7j3kl6jgduz2wck29yl": true,
			},
		},
	},
}

var (
	// wasmd permissions
	WasmPermissionsMap = map[string]bool{
		"switch1qhy9zkhtwxrma0epm0a7ln0lz3hq5vc5p6f3dn": false,
	}

	// wasmd permissions
	WasmInstantiatePermissions = map[string]bool{
		"switch1qhy9zkhtwxrma0epm0a7ln0lz3hq5vc5p6f3dn": false,
	}

	// wasmd permissions
	WasmExecutePermissions = map[string]bool{
		"switch1jgnk2mg88m57csrmrlrd6c3qe4lag3e33y2f3k": true,
		"switch1qhy9zkhtwxrma0epm0a7ln0lz3hq5vc5p6f3dn": false,
		"switch1khtl8ch2zgay00c47ukvulam3a4faw2500g7lu": true,
	}
)

// isPermitted returns whether or not the given address can deploy wasm contracts on mocknet
func isPermitted(addr string) bool {
	whitelist := map[string]bool{
		"switch1jgnk2mg88m57csrmrlrd6c3qe4lag3e33y2f3k": true,
		"switch1qhy9zkhtwxrma0epm0a7ln0lz3hq5vc5p6f3dn": false,
		"switch1khtl8ch2zgay00c47ukvulam3a4faw2500g7lu": true,
	}
	return whitelist[addr]
}
