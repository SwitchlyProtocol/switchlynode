//go:build mocknet
// +build mocknet

package wasmpermissions

var WasmPermissionsRaw = WasmPermissions{
	Permissions: map[string]WasmPermission{
		// No specific permissions for mocknet - empty map
	},
}

var (
	// wasm permissions for mocknet
	// https://github.com/switchlyprotocol/switchlynode/blob/develop/docs/wasm_permissions.md
	// Add wasm permissions for mocknet
	wasmPermissions = map[string]bool{
		"swtc1qhy9zkhtwxrma0epm0a7ln0lz3hq5vc5wsjsxc": false,
	}

	// wasm permissions for mocknet
	// https://github.com/switchlyprotocol/switchlynode/blob/develop/docs/wasm_permissions.md
	// Add wasm permissions for mocknet
	wasmPermissionsValidators = map[string]bool{
		"swtc1qhy9zkhtwxrma0epm0a7ln0lz3hq5vc5wsjsxc": false,
	}
)

// isPermitted returns whether or not the given address can deploy wasm contracts on mocknet
func isPermitted(addr string) bool {
	whitelist := map[string]bool{
		"swtc1jgnk2mg88m57csrmrlrd6c3qe4lag3e33y2f3k": true,
		"swtc1qhy9zkhtwxrma0epm0a7ln0lz3hq5vc5wsjsxc": false,
		"swtc1khtl8ch2zgay00c47ukvulam3a4faw2500g7lu": true,
	}
	return whitelist[addr]
}
