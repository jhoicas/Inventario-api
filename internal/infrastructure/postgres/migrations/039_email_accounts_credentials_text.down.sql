-- 039 rollback - Restore credential columns to VARCHAR(1024)

ALTER TABLE email_accounts
    ALTER COLUMN app_password TYPE VARCHAR(1024),
    ALTER COLUMN password TYPE VARCHAR(1024);
