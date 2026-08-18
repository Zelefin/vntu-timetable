CREATE TABLE settings (
	key TEXT PRIMARY KEY CHECK (trim(key) <> ''),
	value TEXT NOT NULL,
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) STRICT;

INSERT INTO
	settings (key, value)
VALUES
	('week_offset', '0');
