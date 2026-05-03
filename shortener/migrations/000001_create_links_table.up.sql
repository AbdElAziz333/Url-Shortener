CREATE TABLE IF NOT EXISTS links (
    id UUID NOT NULL PRIMARY KEY,
    code VARCHAR(12) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    user_id UUID NOT NULL,
    custom_alias VARCHAR(50) UNIQUE,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- links
-- Column	Type	Notes
-- idPK	uuid	gen_random_uuid()
-- codeIDX	varchar(12)	unique, base62 encoded
-- original_url	text	not null
-- user_idFKIDX	uuid	→ auth_schema.users.id
-- custom_alias	varchar(50)	nullable, unique
-- expires_at	timestamptz	nullable
-- is_active	boolean	default true
-- created_at	timestamptz	default now()