CREATE TABLE faculties (
	id INTEGER PRIMARY KEY CHECK (id > 0),
	name TEXT NOT NULL CHECK (trim(name) <> ''),
	active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) STRICT;

CREATE TABLE groups (
	id INTEGER PRIMARY KEY CHECK (id > 0),
	faculty_id INTEGER NOT NULL REFERENCES faculties (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
	name TEXT NOT NULL CHECK (trim(name) <> ''),
	active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
	schedule_updated_at TEXT,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) STRICT;

CREATE INDEX groups_faculty_active_name_idx ON groups (faculty_id, active, name);

CREATE TABLE lessons (
	id INTEGER PRIMARY KEY,
	group_id INTEGER NOT NULL REFERENCES groups (id) ON UPDATE RESTRICT ON DELETE CASCADE,
	week_number INTEGER NOT NULL CHECK (week_number IN (1, 2)),
	day_number INTEGER NOT NULL CHECK (day_number BETWEEN 0 AND 6),
	lesson_number INTEGER NOT NULL CHECK (lesson_number > 0),
	subgroup INTEGER NOT NULL CHECK (subgroup IN (0, 1, 2)),
	subject TEXT NOT NULL CHECK (trim(subject) <> ''),
	lesson_type TEXT NOT NULL CHECK (trim(lesson_type) <> ''),
	teacher_name TEXT NOT NULL,
	room TEXT NOT NULL,
	start_minute INTEGER NOT NULL CHECK (start_minute BETWEEN 0 AND 1439),
	end_minute INTEGER NOT NULL CHECK (end_minute BETWEEN 1 AND 1440),
	CHECK (start_minute < end_minute)
) STRICT;

CREATE UNIQUE INDEX lessons_group_schedule_idx ON lessons (
	group_id,
	week_number,
	day_number,
	lesson_number,
	subgroup
);

CREATE TABLE users (
	telegram_user_id INTEGER PRIMARY KEY CHECK (telegram_user_id > 0),
	group_id INTEGER REFERENCES groups (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
	subgroup INTEGER,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	CHECK (
		(
			group_id IS NULL
			AND subgroup IS NULL
		)
		OR (
			group_id IS NOT NULL
			AND subgroup IN (0, 1, 2)
		)
	)
) STRICT;
