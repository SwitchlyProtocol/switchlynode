#![cfg(test)]
use super::*;
use soroban_sdk::{
    testutils::{Address as _, Events, Ledger}, 
    token::{Client as TokenClient, StellarAssetClient}, 
    Address, Env, String, vec
};

fn create_token_contract<'a>(e: &Env, admin: &Address) -> (Address, TokenClient<'a>, StellarAssetClient<'a>) {
    let stellar_asset_contract = e.register_stellar_asset_contract_v2(admin.clone());
    let contract_address = stellar_asset_contract.address();
    (
        contract_address.clone(), 
        TokenClient::new(e, &contract_address),
        StellarAssetClient::new(e, &contract_address)
    )
}

#[test]
fn test_basic_deposit() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let from = Address::generate(&env);
    let vault = Address::generate(&env);
    
    // Create a token contract
    let (asset, token_client, stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Mint tokens to the from address
    stellar_asset_client.mint(&from, &1000);
    
    // Test deposit
    let amount = 1000i128;
    let memo = String::from_str(&env, "SWAP:BTC.BTC:bc1qxyz");
    
    client.deposit(&from, &vault, &asset, &amount, &memo);
    
    // Verify token was transferred
    assert_eq!(token_client.balance(&from), 0);
    assert_eq!(token_client.balance(&vault), 1000);
}

#[test]
fn test_deposit_with_expiry() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let from = Address::generate(&env);
    let vault = Address::generate(&env);
    
    // Create a token contract
    let (asset, token_client, stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Mint tokens to the from address
    stellar_asset_client.mint(&from, &1000);
    
    // Test deposit with future expiry
    let amount = 1000i128;
    let memo = String::from_str(&env, "SWAP:BTC.BTC:bc1qxyz");
    let expiry = env.ledger().timestamp() + 3600; // 1 hour from now
    
    client.deposit_with_expiry(&from, &vault, &asset, &amount, &memo, &expiry);
    
    // Verify token was transferred
    assert_eq!(token_client.balance(&from), 0);
    assert_eq!(token_client.balance(&vault), 1000);
}

#[test]
#[should_panic(expected = "Transaction expired")]
fn test_deposit_with_expiry_expired() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let from = Address::generate(&env);
    let vault = Address::generate(&env);
    
    // Create a token contract
    let (asset, _token_client, _stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Test deposit with past expiry - should panic
    let amount = 1000i128;
    let memo = String::from_str(&env, "SWAP:BTC.BTC:bc1qxyz");
    
    // Set a timestamp in the past by setting current time to a higher value
    env.ledger().with_mut(|ledger| {
        ledger.timestamp = 1000; // Set current time to 1000
    });
    
    let expiry = 999; // Set expiry to 999 (in the past)
    
    client.deposit_with_expiry(&from, &vault, &asset, &amount, &memo, &expiry);
}

#[test]
fn test_transfer_out() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let vault = Address::generate(&env);
    let recipient = Address::generate(&env);
    
    // Create a token contract
    let (asset, token_client, stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Mint tokens to the vault address
    stellar_asset_client.mint(&vault, &1000);
    
    // Test transfer out (called by vault)
    let amount = 500i128;
    let memo = String::from_str(&env, "OUT:HASH");
    
    client.transfer_out(&vault, &recipient, &asset, &amount, &memo);
    
    // Verify token was transferred
    assert_eq!(token_client.balance(&vault), 500);
    assert_eq!(token_client.balance(&recipient), 500);
}

#[test]
fn test_transfer_allowance() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let old_vault = Address::generate(&env);
    let new_vault = Address::generate(&env);
    
    // Create a token contract
    let (asset, token_client, stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Mint tokens to the old vault
    stellar_asset_client.mint(&old_vault, &1000);
    
    // Test vault rotation using return_vault_assets (simplified approach)
    let amount = 1000i128;
    let memo = String::from_str(&env, "MIGRATE:VAULT");
    
    let assets = vec![&env, asset.clone()];
    let amounts = vec![&env, amount];
    client.return_vault_assets(&old_vault, &new_vault, &assets, &amounts, &memo);
    
    // Verify token was transferred
    assert_eq!(token_client.balance(&old_vault), 0);
    assert_eq!(token_client.balance(&new_vault), 1000);
}

#[test]
fn test_return_vault_assets() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let old_vault = Address::generate(&env);
    let new_vault = Address::generate(&env);
    
    // Create token contracts
    let (asset1, token_client1, stellar_asset_client1) = create_token_contract(&env, &admin);
    let (asset2, token_client2, stellar_asset_client2) = create_token_contract(&env, &admin);
    
    // Mint tokens to the old vault
    stellar_asset_client1.mint(&old_vault, &1000);
    stellar_asset_client2.mint(&old_vault, &2000);
    
    // Test return vault assets
    let assets = vec![&env, asset1.clone(), asset2.clone()];
    let amounts = vec![&env, 1000i128, 2000i128];
    let memo = String::from_str(&env, "CHURN:COMPLETE");
    
    client.return_vault_assets(&old_vault, &new_vault, &assets, &amounts, &memo);
    
    // Verify tokens were transferred
    assert_eq!(token_client1.balance(&old_vault), 0);
    assert_eq!(token_client1.balance(&new_vault), 1000);
    assert_eq!(token_client2.balance(&old_vault), 0);
    assert_eq!(token_client2.balance(&new_vault), 2000);
}

#[test]
#[should_panic(expected = "Assets and amounts array length mismatch")]
fn test_return_vault_assets_length_mismatch() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let old_vault = Address::generate(&env);
    let new_vault = Address::generate(&env);
    let asset1 = Address::generate(&env);
    let asset2 = Address::generate(&env);
    
    // Test with mismatched arrays - should panic
    let assets = vec![&env, asset1, asset2];
    let amounts = vec![&env, 1000i128]; // Only one amount for two assets
    let memo = String::from_str(&env, "CHURN:COMPLETE");
    
    client.return_vault_assets(&old_vault, &new_vault, &assets, &amounts, &memo);
}

#[test]
fn test_version() {
    let env = Env::default();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    let version = client.version();
    assert_eq!(version, String::from_str(&env, "3.0.0-stateless"));
}

#[test]
fn test_multiple_deposits_different_vaults() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let from = Address::generate(&env);
    let vault1 = Address::generate(&env);
    let vault2 = Address::generate(&env);
    
    // Create a token contract
    let (asset, token_client, stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Mint tokens to the from address
    stellar_asset_client.mint(&from, &3000);
    
    let amount1 = 1000i128;
    let amount2 = 2000i128;
    
    let memo1 = String::from_str(&env, "SWAP:ETH.ETH:0x123");
    let memo2 = String::from_str(&env, "SWAP:BTC.BTC:0x456");
    
    // Make deposits to different vaults
    client.deposit(&from, &vault1, &asset, &amount1, &memo1);
    client.deposit(&from, &vault2, &asset, &amount2, &memo2);
    
    // Verify tokens were transferred correctly
    assert_eq!(token_client.balance(&from), 0);
    assert_eq!(token_client.balance(&vault1), 1000);
    assert_eq!(token_client.balance(&vault2), 2000);
}

#[test]
fn test_multi_asset_support() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let from = Address::generate(&env);
    let vault = Address::generate(&env);
    
    // Create token contracts
    let (asset1, token_client1, stellar_asset_client1) = create_token_contract(&env, &admin);
    let (asset2, token_client2, stellar_asset_client2) = create_token_contract(&env, &admin);
    
    // Mint tokens to the from address
    stellar_asset_client1.mint(&from, &1000);
    stellar_asset_client2.mint(&from, &2000);
    
    let amount1 = 1000i128;
    let amount2 = 2000i128;
    
    let memo1 = String::from_str(&env, "SWAP:ETH.ETH:0x123");
    let memo2 = String::from_str(&env, "SWAP:BTC.BTC:0x456");
    
    // Deposit different assets to same vault
    client.deposit(&from, &vault, &asset1, &amount1, &memo1);
    client.deposit(&from, &vault, &asset2, &amount2, &memo2);
    
    // Verify tokens were transferred
    assert_eq!(token_client1.balance(&from), 0);
    assert_eq!(token_client1.balance(&vault), 1000);
    assert_eq!(token_client2.balance(&from), 0);
    assert_eq!(token_client2.balance(&vault), 2000);
}

#[test]
fn test_vault_rotation_scenario() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Simulate vault rotation: old_vault -> new_vault
    let admin = Address::generate(&env);
    let old_vault = Address::generate(&env);
    let new_vault = Address::generate(&env);
    
    // Create token contracts
    let (asset1, token_client1, stellar_asset_client1) = create_token_contract(&env, &admin);
    let (asset2, token_client2, stellar_asset_client2) = create_token_contract(&env, &admin);
    
    // Mint tokens to the old vault
    stellar_asset_client1.mint(&old_vault, &1500);
    stellar_asset_client2.mint(&old_vault, &3000);
    
    // Transfer individual assets using return_vault_assets (simplified approach)
    let assets1 = vec![&env, asset1.clone()];
    let amounts1 = vec![&env, 1000i128];
    client.return_vault_assets(&old_vault, &new_vault, &assets1, &amounts1, &String::from_str(&env, "ROTATE:ASSET1"));
    
    let assets2 = vec![&env, asset2.clone()];
    let amounts2 = vec![&env, 2000i128];
    client.return_vault_assets(&old_vault, &new_vault, &assets2, &amounts2, &String::from_str(&env, "ROTATE:ASSET2"));
    
    // Or transfer all assets at once
    let assets = vec![&env, asset1.clone(), asset2.clone()];
    let amounts = vec![&env, 500i128, 1000i128];
    client.return_vault_assets(&old_vault, &new_vault, &assets, &amounts, &String::from_str(&env, "ROTATE:ALL"));
    
    // Verify all tokens were transferred
    assert_eq!(token_client1.balance(&old_vault), 0);
    assert_eq!(token_client1.balance(&new_vault), 1500);
    assert_eq!(token_client2.balance(&old_vault), 0);
    assert_eq!(token_client2.balance(&new_vault), 3000);
}

#[test]
fn test_large_amount_deposit() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let from = Address::generate(&env);
    let vault = Address::generate(&env);
    
    // Create a token contract
    let (asset, token_client, stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Test with large amount (close to i128::MAX but reasonable)
    let large_amount = 1_000_000_000_000_000_000i128; // 1 quintillion
    
    // Mint tokens to the from address
    stellar_asset_client.mint(&from, &large_amount);
    
    let memo = String::from_str(&env, "SWAP:ETH.ETH:0x123");
    
    // Call deposit with large amount
    client.deposit(&from, &vault, &asset, &large_amount, &memo);
    
    // Verify token was transferred
    assert_eq!(token_client.balance(&from), 0);
    assert_eq!(token_client.balance(&vault), large_amount);
}

#[test]
fn test_event_emission() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let from = Address::generate(&env);
    let vault = Address::generate(&env);
    
    // Create a token contract
    let (asset, token_client, stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Mint tokens to the from address
    stellar_asset_client.mint(&from, &1000);
    
    let amount = 1000i128;
    let memo = String::from_str(&env, "SWAP:ETH.ETH:0x123");
    
    // Call deposit
    client.deposit(&from, &vault, &asset, &amount, &memo);
    
    // Verify event was emitted
    let events = env.events().all();
    
    // Event should be emitted from our contract
    let has_contract_event = events.iter().any(|event| {
        event.0 == contract_id
    });
    assert!(has_contract_event);
    
    // Also verify token was transferred
    assert_eq!(token_client.balance(&from), 0);
    assert_eq!(token_client.balance(&vault), 1000);
}

#[test]
fn test_no_initialization_required() {
    let env = Env::default();
    env.mock_all_auths();
    
    let contract_id = env.register_contract(None, SwitchlyRouter);
    let client = SwitchlyRouterClient::new(&env, &contract_id);
    
    // Create test addresses
    let admin = Address::generate(&env);
    let from = Address::generate(&env);
    let vault = Address::generate(&env);
    
    // Create a token contract
    let (asset, token_client, stellar_asset_client) = create_token_contract(&env, &admin);
    
    // Mint tokens to the from address
    stellar_asset_client.mint(&from, &1000);
    
    // Should be able to call deposit immediately without initialization
    let amount = 1000i128;
    let memo = String::from_str(&env, "SWAP:ETH.ETH:0x123");
    
    client.deposit(&from, &vault, &asset, &amount, &memo);
    
    // Should be able to call version immediately
    let version = client.version();
    assert_eq!(version, String::from_str(&env, "3.0.0-stateless"));
    
    // Verify token was transferred (no initialization was required)
    assert_eq!(token_client.balance(&from), 0);
    assert_eq!(token_client.balance(&vault), 1000);
} 