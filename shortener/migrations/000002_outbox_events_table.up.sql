CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID NOT NULL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);

-- outbox_events
-- Column	Type	Notes
-- idPK	uuid	gen_random_uuid()
-- event_type	varchar(50)	e.g. link.created
-- payload	jsonb	full event body
-- statusIDX	varchar(20)	pending · published · failed
-- created_at	timestamptz	default now()
-- published_at	timestamptz	nullable, set by worker