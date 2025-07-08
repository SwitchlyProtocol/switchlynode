#![no_std]
use soroban_sdk::{
    contract, contractimpl, contracttype, token, Address, Env, String, Symbol, Vec, log
};

// Simple event structure matching ETH router pattern
#[contracttype]
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DepositEvent {
    pub vault: Address,      // Vault address receiving the deposit
    pub from: Address,       // User making the deposit
    pub asset: Address,      // Asset being deposited
    pub amount: i128,        // Amount being deposited
    pub memo: String,        // SwitchlyProtocol memo
}

#[contracttype]
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TransferOutEvent {
    pub vault: Address,      // Vault address sending the transfer
    pub to: Address,         // Recipient address
    pub asset: Address,      // Asset being transferred
    pub amount: i128,        // Amount being transferred
    pub memo: String,        // SwitchlyProtocol memo
}

#[contracttype]
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TransferAllowanceEvent {
    pub old_vault: Address,  // Old vault address
    pub new_vault: Address,  // New vault address
    pub asset: Address,      // Asset being transferred
    pub amount: i128,        // Amount being transferred
    pub memo: String,        // SwitchlyProtocol memo
}

#[contracttype]
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct VaultReturnEvent {
    pub old_vault: Address,  // Old vault address
    pub new_vault: Address,  // New vault address
    pub assets: Vec<Address>, // Assets being returned
    pub amounts: Vec<i128>,  // Amounts being returned
    pub memo: String,        // SwitchlyProtocol memo
}

// Switchly Router Contract - Stellar Implementation
// Follows EVM pattern: no initialization required, vault addresses passed per transaction
#[contract]
pub struct SwitchlyRouter;

#[contractimpl]
impl SwitchlyRouter {

    /// Deposit assets to a vault (called by users)
    /// This is the main entry point for users to deposit assets
    pub fn deposit(
        env: Env,
        from: Address,       // User making the deposit
        vault: Address,
        asset: Address,
        amount: i128,
        memo: String,
    ) {
        // Require authorization from the user making the deposit
        from.require_auth();
        
        // Transfer the asset from the user to the vault
        let token_client = token::Client::new(&env, &asset);
        token_client.transfer(&from, &vault, &amount);
        
        // Emit deposit event
        env.events().publish(
            (Symbol::new(&env, "deposit"),),
            DepositEvent {
                vault: vault.clone(),
                from: from.clone(),
                asset: asset.clone(),
                amount,
                memo: memo.clone(),
            }
        );
        
        log!(&env, "Deposit: from={}, vault={}, asset={}, amount={}, memo={}", from, vault, asset, amount, memo);
    }

    /// Deposit assets with expiry (called by users)
    pub fn deposit_with_expiry(
        env: Env,
        from: Address,       // User making the deposit
        vault: Address,
        asset: Address,
        amount: i128,
        memo: String,
        expiry: u64,
    ) {
        // Check if the transaction has expired
        let current_time = env.ledger().timestamp();
        if current_time > expiry {
            panic!("Transaction expired");
        }
        
        // Call the regular deposit function
        Self::deposit(env, from, vault, asset, amount, memo);
    }

    /// Transfer assets out of a vault (called by vaults)
    pub fn transfer_out(
        env: Env,
        vault: Address,      // Vault making the transfer
        to: Address,
        asset: Address,
        amount: i128,
        memo: String,
    ) {
        // Require authorization from the vault making the transfer
        vault.require_auth();
        
        // Transfer the asset from the vault to the recipient
        let token_client = token::Client::new(&env, &asset);
        token_client.transfer(&vault, &to, &amount);
        
        // Emit transfer out event
        env.events().publish(
            (Symbol::new(&env, "transfer_out"),),
            TransferOutEvent {
                vault: vault.clone(),
                to: to.clone(),
                asset: asset.clone(),
                amount,
                memo: memo.clone(),
            }
        );
        
        log!(&env, "Transfer out: vault={}, to={}, asset={}, amount={}, memo={}", vault, to, asset, amount, memo);
    }

    /// Transfer allowance between vaults (vault rotation)
    pub fn transfer_allowance(
        env: Env,
        old_vault: Address,
        new_vault: Address,
        asset: Address,
        amount: i128,
        memo: String,
    ) {
        // Require authorization from the old vault
        old_vault.require_auth();
        
        // Transfer the asset from old vault to new vault
        let token_client = token::Client::new(&env, &asset);
        token_client.transfer(&old_vault, &new_vault, &amount);
        
        // Emit transfer allowance event
        env.events().publish(
            (Symbol::new(&env, "transfer_allowance"),),
            TransferAllowanceEvent {
                old_vault: old_vault.clone(),
                new_vault: new_vault.clone(),
                asset: asset.clone(),
                amount,
                memo: memo.clone(),
            }
        );
        
        log!(&env, "Transfer allowance: old_vault={}, new_vault={}, asset={}, amount={}, memo={}", 
             old_vault, new_vault, asset, amount, memo);
    }

    /// Return multiple assets from old vault to new vault
    pub fn return_vault_assets(
        env: Env,
        old_vault: Address,
        new_vault: Address,
        assets: Vec<Address>,
        amounts: Vec<i128>,
        memo: String,
    ) {
        // Validate that assets and amounts arrays have the same length
        if assets.len() != amounts.len() {
            panic!("Assets and amounts length mismatch");
        }
        
        // Require authorization from the old vault
        old_vault.require_auth();
        
        // Transfer each asset from old vault to new vault
        for i in 0..assets.len() {
            let asset = assets.get(i).unwrap();
            let amount = amounts.get(i).unwrap();
            
            let token_client = token::Client::new(&env, &asset);
            token_client.transfer(&old_vault, &new_vault, &amount);
        }
        
        // Emit vault return event
        env.events().publish(
            (Symbol::new(&env, "return_vault_assets"),),
            VaultReturnEvent {
                old_vault: old_vault.clone(),
                new_vault: new_vault.clone(),
                assets: assets.clone(),
                amounts: amounts.clone(),
                memo: memo.clone(),
            }
        );
        
        log!(&env, "Return vault assets: old_vault={}, new_vault={}, assets_count={}, memo={}", 
             old_vault, new_vault, assets.len(), memo);
    }

    /// Get contract version
    pub fn version(env: Env) -> String {
        String::from_str(&env, "1.0.0")
    }
}

#[cfg(test)]
mod test; 