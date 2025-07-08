//go:build mocknet
// +build mocknet

package wasmpermissions

var WasmPermissionsRaw = WasmPermissions{
	Permissions: map[string]WasmPermission{
		"a8f1a38aa518864169e30ab482ea86558a817982a030b8888ea6dfa0cd700128": {
			Origin: "https://thorchain.org",
			Deployers: map[string]bool{
				"swtc1jgnk2mg88m57csrmrlrd6c3qe4lag3e33y2f3k": true,
				"swtc1tdfqy34uptx207scymqsy4k5uzfmry5sf7z3dw": false,
				"swtc1khtl8ch2zgay00c47ukvulam3a4faw2500g7lu": true,
			},
		},
	},
}

// isPermitted returns whether or not the given address can deploy wasm contracts on mocknet
func isPermitted(addr string) bool {
	whitelist := map[string]bool{
		"swtc1jgnk2mg88m57csrmrlrd6c3qe4lag3e33y2f3k": true,
		"swtc1tdfqy34uptx207scymqsy4k5uzfmry5sf7z3dw": false,
		"swtc1khtl8ch2zgay00c47ukvulam3a4faw2500g7lu": true,
	}
	return whitelist[addr]
}
