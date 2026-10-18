CREATE TABLE IF NOT EXISTS sla_rules (
    id BIGSERIAL PRIMARY KEY,
    topic_id BIGINT NOT NULL REFERENCES ticket_topics(id),
    priority_id INT NOT NULL REFERENCES ticket_priorities(id),
    response_time_minutes INT NOT NULL CHECK (response_time_minutes > 0),
    resolution_time_minutes INT NOT NULL CHECK (resolution_time_minutes > 0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_sla_rules_topic_priority UNIQUE (topic_id, priority_id)
);

CREATE INDEX IF NOT EXISTS idx_sla_rules_topic_priority ON sla_rules(topic_id, priority_id);

INSERT INTO sla_rules (topic_id, priority_id, response_time_minutes, resolution_time_minutes)
SELECT topic.id, rule.priority_id, rule.response_minutes, rule.resolution_minutes
FROM ticket_topics AS topic
CROSS JOIN (VALUES
    (1, 480, 2880),
    (2, 240, 1440),
    (3, 60, 480),
    (4, 15, 120)
) AS rule(priority_id, response_minutes, resolution_minutes)
ON CONFLICT (topic_id, priority_id) DO NOTHING;

ALTER TABLE tickets
    ADD COLUMN response_deadline TIMESTAMP,
    ADD COLUMN resolution_deadline TIMESTAMP,
    ADD COLUMN first_response_at TIMESTAMP,
    ADD COLUMN resolved_at TIMESTAMP;

UPDATE tickets AS ticket
SET response_deadline = ticket.created_at + make_interval(mins => rule.response_time_minutes),
    resolution_deadline = ticket.created_at + make_interval(mins => rule.resolution_time_minutes)
FROM sla_rules AS rule
WHERE rule.topic_id = ticket.topic_id AND rule.priority_id = ticket.priority_id;

ALTER TABLE tickets
    ALTER COLUMN response_deadline SET NOT NULL,
    ALTER COLUMN resolution_deadline SET NOT NULL;

CREATE OR REPLACE FUNCTION set_ticket_sla_deadlines()
RETURNS TRIGGER AS $$
DECLARE
    selected_rule sla_rules%ROWTYPE;
BEGIN
    IF NEW.response_deadline IS NULL OR NEW.resolution_deadline IS NULL THEN
        SELECT * INTO STRICT selected_rule
        FROM sla_rules
        WHERE topic_id = NEW.topic_id AND priority_id = NEW.priority_id;

        NEW.response_deadline := COALESCE(
            NEW.response_deadline,
            NEW.created_at + make_interval(mins => selected_rule.response_time_minutes)
        );
        NEW.resolution_deadline := COALESCE(
            NEW.resolution_deadline,
            NEW.created_at + make_interval(mins => selected_rule.resolution_time_minutes)
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_ticket_sla_deadlines_before_insert
    BEFORE INSERT ON tickets
    FOR EACH ROW EXECUTE FUNCTION set_ticket_sla_deadlines();

CREATE INDEX IF NOT EXISTS idx_tickets_response_deadline ON tickets(response_deadline)
    WHERE first_response_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tickets_resolution_deadline ON tickets(resolution_deadline)
    WHERE resolved_at IS NULL;
