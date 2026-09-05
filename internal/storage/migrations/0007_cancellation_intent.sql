-- Migration 0007: Cancellation intent durability (RC-2).
-- Persist cancellation reason and compensation intent across process restarts.
ALTER TABLE workflow_instances ADD COLUMN cancellation_reason TEXT;
ALTER TABLE workflow_instances ADD COLUMN cancellation_compensate INTEGER NOT NULL DEFAULT 0;
