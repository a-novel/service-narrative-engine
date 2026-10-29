-- Dropping the tables deletes every generation and its calls.
DROP TABLE IF EXISTS generation_calls;

DROP INDEX IF EXISTS generations_project_id_idx;

DROP TABLE IF EXISTS generations;

DROP TYPE IF EXISTS generation_call_status;

DROP TYPE IF EXISTS generation_kind;
