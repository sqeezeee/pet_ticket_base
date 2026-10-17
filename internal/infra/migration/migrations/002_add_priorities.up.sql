CREATE TABLE IF NOT EXISTS ticket_priorities (
    id INT PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT
);

INSERT INTO ticket_priorities (id, name, description) VALUES
(1, 'low', 'Низкий приоритет'),
(2, 'medium', 'Средний приоритет'),
(3, 'high', 'Высокий приоритет'),
(4, 'critical', 'Критический приоритет')
ON CONFLICT (id) DO NOTHING;

ALTER TABLE tickets
    ADD COLUMN priority_id INT NOT NULL DEFAULT 2;

ALTER TABLE tickets
    ADD CONSTRAINT fk_tickets_priority_id
    FOREIGN KEY (priority_id) REFERENCES ticket_priorities(id);

CREATE INDEX IF NOT EXISTS idx_tickets_priority_id ON tickets(priority_id);
