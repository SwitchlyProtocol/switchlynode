#![no_std]
use soroban_sdk::{
    contract, contractimpl, contracttype, token, Address, Env, String, Symbol, Vec, log
};

// Event structures
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
pub struct VaultReturnEvent {
    pub old_vault: Address,  // Old vault address
    pub new_vault: Address,  // New vault address
    pub assets: Vec<Address>, // Assets being returned
    pub amounts: Vec<i128>,  // Amounts being returned
    pub memo: String,        // SwitchlyProtocol memo
}

/// Switchly Router Contract - Stellar Implementation
/// 
/// A stateless router that forwards all assets directly to vaults:
/// - All assets (XLM, tokens, etc.) are transferred directly to vaults
/// - No allowance tracking or router custody
/// - Vaults maintain direct control over their assets
#[contract]
pub struct SwitchlyRouter;

#[contractimpl]
impl SwitchlyRouter {

    /// Deposit assets directly to vault
    /// 
    /// Transfers assets from user to vault with immediate custody:
    /// - All asset types supported (XLM, tokens, etc.)
    /// - Direct transfer with no intermediate storage
    /// - Vault receives immediate control of deposited assets
    /// 
    /// # Arguments
    /// * `from` - User address making the deposit
    /// * `vault` - Target vault address
    /// * `asset` - Asset contract address
    /// * `amount` - Deposit amount in asset's native decimals
    /// * `memo` - SwitchlyProtocol transaction memo
    pub fn deposit(
        env: Env,
        from: Address,
        vault: Address,
        asset: Address,
        amount: i128,
        memo: String,
    ) {
        from.require_auth();
        
        // Transfer asset directly to vault
        let token_client = token::Client::new(&env, &asset);
        token_client.transfer(&from, &vault, &amount);
        
        // Emit standardized deposit event
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
        
        log!(&env, "Deposit completed: from={}, vault={}, asset={}, amount={}, memo={}", 
             from, vault, asset, amount, memo);
    }


    /// Deposit assets with expiry validation
    /// 
    /// Extends the standard deposit function with time-based expiration checking.
    /// Prevents execution of stale transactions that exceed their validity window.
    /// 
    /// # Arguments
    /// * `from` - User address making the deposit
    /// * `vault` - Target vault address
    /// * `asset` - Asset contract address
    /// * `amount` - Deposit amount in asset's native decimals
    /// * `memo` - SwitchlyProtocol transaction memo
    /// * `expiry` - Unix timestamp after which transaction becomes invalid
    pub fn deposit_with_expiry(
        env: Env,
        from: Address,
        vault: Address,
        asset: Address,
        amount: i128,
        memo: String,
        expiry: u64,
    ) {
        // Validate transaction has not expired
        let current_time = env.ledger().timestamp();
        if current_time > expiry {
            panic!("Transaction expired");
        }
        
        // Execute deposit with expiry validation
        Self::deposit(env, from, vault, asset, amount, memo);
    }

    /// Transfer assets out of a vault
    /// 
    /// Authorizes vault to transfer assets to recipients:
    /// - Vault authorization required
    /// - Direct transfer from vault to recipient
    /// - No intermediate custody or storage
    /// 
    /// # Arguments
    /// * `vault` - Vault address authorizing the transfer
    /// * `to` - Recipient address
    /// * `asset` - Asset contract address
    /// * `amount` - Transfer amount in asset's native decimals
    /// * `memo` - SwitchlyProtocol transaction memo
    pub fn transfer_out(
        env: Env,
        vault: Address,
        to: Address,
        asset: Address,
        amount: i128,
        memo: String,
    ) {
        vault.require_auth();
        
        // Transfer asset from vault to recipient
        let token_client = token::Client::new(&env, &asset);
        token_client.transfer(&vault, &to, &amount);
        
        // Emit standardized transfer out event
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
        
        log!(&env, "Transfer out completed: vault={}, to={}, asset={}, amount={}, memo={}", 
             vault, to, asset, amount, memo);
    }

    /// Return multiple assets from old vault to new vault during rotation
    /// 
    /// Batch transfer operation for vault migration.
    /// Transfers multiple assets from source vault to destination vault.
    /// 
    /// # Arguments
    /// * `old_vault` - Source vault address authorizing the transfers
    /// * `new_vault` - Destination vault address
    /// * `assets` - Array of asset contract addresses
    /// * `amounts` - Array of transfer amounts (must match assets array length)
    /// * `memo` - SwitchlyProtocol transaction memo
    pub fn return_vault_assets(
        env: Env,
        old_vault: Address,
        new_vault: Address,
        assets: Vec<Address>,
        amounts: Vec<i128>,
        memo: String,
    ) {
        // Validate input arrays have matching lengths
        if assets.len() != amounts.len() {
            panic!("Assets and amounts array length mismatch");
        }
        
        old_vault.require_auth();
        
        // Transfer each asset from old vault to new vault
        for i in 0..assets.len() {
            let asset = assets.get(i).unwrap();
            let amount = amounts.get(i).unwrap();
            
            let token_client = token::Client::new(&env, &asset);
            token_client.transfer(&old_vault, &new_vault, &amount);
        }
        
        // Emit standardized vault return event
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
        
        log!(&env, "Return vault assets completed: old_vault={}, new_vault={}, assets_count={}, memo={}", 
             old_vault, new_vault, assets.len(), memo);
    }

    /// Get contract version
    /// 
    /// Returns the current version of the SwitchlyRouter contract.
    /// 
    /// # Returns
    /// * `String` - Contract version string
    pub fn version(env: Env) -> String {
        String::from_str(&env, "3.0.0-stateless")
    }
}

#[cfg(test)]
mod test; 