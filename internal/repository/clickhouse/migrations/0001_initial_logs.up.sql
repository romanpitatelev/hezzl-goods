CREATE TABLE IF NOT EXISTS goods_logs
(
    id UInt32,
    project_id UInt32,
    name String,
    description Nullable(String),
    priority UInt32,
    removed UInt8,
    operation String,
    event_time DateTime DEFAULT now(),

    INDEX id_idx id TYPE minmax GRANULARITY 3,
    INDEX project_id_idx project_id TYPE minmax GRANULARITY 3,
    INDEX name_idx name TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 3,

    event_date Date MATERIALIZED toDate(event_time)
) ENGINE = MergeTree()
ORDER BY (event_time, id)
PARTITION BY (event_time, id, project_id)
SETTINGS index_granularity = 8192
