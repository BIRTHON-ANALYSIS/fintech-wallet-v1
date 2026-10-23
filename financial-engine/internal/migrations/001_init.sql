```sql
CREATE TABLE wallets (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    currency VARCHAR(3) NOT NULL
        CHECK (currency IN ('NGN', 'USD', 'EUR')),
    balance_minor BIGINT NOT NULL DEFAULT 0
        CHECK (balance_minor >= 0),
    status TEXT NOT NULL
        CHECK (status IN ('active', 'frozen', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);

CREATE TABLE financial_transactions (
    id TEXT PRIMARY KEY,
    reference TEXT NOT NULL UNIQUE,
    idempotency_key TEXT NOT NULL,
    type TEXT NOT NULL CHECK (
        type IN (
            'deposit',
            'withdrawal',
            'transfer',
            'payment',
            'refund',
            'exchange',
            'bill_payment'
        )
    ),
    status TEXT NOT NULL CHECK (
        status IN (
            'pending',
            'processing',
            'succeeded',
            'failed',
            'reversed'
        )
    ),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency VARCHAR(3) NOT NULL
        CHECK (currency IN ('NGN', 'USD', 'EUR')),
    source_wallet_id TEXT REFERENCES wallets(id),
    destination_wallet_id TEXT REFERENCES wallets(id),
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_source_created
    ON financial_transactions(source_wallet_id, created_at DESC);

CREATE INDEX idx_transactions_destination_created
    ON financial_transactions(destination_wallet_id, created_at DESC);

CREATE TABLE ledger_entries (
    id TEXT PRIMARY KEY,
    transaction_id TEXT NOT NULL
        REFERENCES financial_transactions(id),
    account_id TEXT NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('debit', 'credit')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency VARCHAR(3) NOT NULL
        CHECK (currency IN ('NGN', 'USD', 'EUR')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_transaction
    ON ledger_entries(transaction_id);

CREATE INDEX idx_ledger_account_created
    ON ledger_entries(account_id, created_at DESC);

CREATE TABLE idempotency_records (
    idempotency_key TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    transaction_id TEXT REFERENCES financial_transactions(id),
    status TEXT NOT NULL CHECK (
        status IN ('processing', 'completed', 'failed')
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_idempotency_expiry
    ON idempotency_records(expires_at);
