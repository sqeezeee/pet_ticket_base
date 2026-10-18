DROP INDEX IF EXISTS idx_tickets_resolution_deadline;
DROP INDEX IF EXISTS idx_tickets_response_deadline;
DROP TRIGGER IF EXISTS set_ticket_sla_deadlines_before_insert ON tickets;
DROP FUNCTION IF EXISTS set_ticket_sla_deadlines();
ALTER TABLE tickets
    DROP COLUMN IF EXISTS resolved_at,
    DROP COLUMN IF EXISTS first_response_at,
    DROP COLUMN IF EXISTS resolution_deadline,
    DROP COLUMN IF EXISTS response_deadline;
DROP INDEX IF EXISTS idx_sla_rules_topic_priority;
DROP TABLE IF EXISTS sla_rules;
