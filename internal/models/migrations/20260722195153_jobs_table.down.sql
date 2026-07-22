-- Dropping the table takes its indexes and its autovacuum settings with it, so there is nothing to
-- unwind first. Every queued and settled job is lost, which is the intended reversal: the table is
-- the only record of that work.
DROP TABLE IF EXISTS jobs;
