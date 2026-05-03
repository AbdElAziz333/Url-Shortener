CREATE TABLE IF NOT EXISTS link_stats_daily (
    id UUID NOT NULL PRIMARY KEY,
    link_code VARCHAR(12) NOT NULL UNIQUE,
    date ,
    click_count,
    unique_ips
);

-- link_stats_daily
-- Column	Type	Notes
-- idPK	uuid	gen_random_uuid()
-- link_codeIDX	varchar(12)	denormalized — no FK cross-schema
-- dateIDX	date	aggregate bucket
-- click_count	integer	incremented by consumer
-- unique_ips	integer	HyperLogLog approximation