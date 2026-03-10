-- 022 - Migración a roles múltiples en users (TEXT[])

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS roles TEXT[] NOT NULL DEFAULT ARRAY['sales'];

-- Copiar el rol actual (legacy) al nuevo arreglo de roles.
UPDATE users SET roles = ARRAY[role] WHERE role IS NOT NULL;

-- Eliminar columna legacy role.
ALTER TABLE users
    DROP COLUMN IF EXISTS role;

