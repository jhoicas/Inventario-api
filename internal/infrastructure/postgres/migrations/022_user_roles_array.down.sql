-- Revertir 022 - roles múltiples en users

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role VARCHAR(20);

-- Tomar el primer rol del arreglo como role legacy.
UPDATE users SET role = roles[1] WHERE role IS NULL AND roles IS NOT NULL AND array_length(roles, 1) >= 1;

ALTER TABLE users
    DROP COLUMN IF EXISTS roles;

