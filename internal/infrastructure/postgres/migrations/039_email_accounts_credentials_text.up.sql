-- 039 - Expand email credential/token storage to TEXT

ALTER TABLE email_accounts
    ALTER COLUMN password TYPE TEXT,
    ALTER COLUMN app_password TYPE TEXT;
