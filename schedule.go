package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var errGroupNotFound = errors.New("group not found")

func replaceGroupLessons(
	ctx context.Context,
	db *sql.DB,
	groupID int64,
	lessons []Lesson,
	scheduleUpdatedAt time.Time,
) error {
	if scheduleUpdatedAt.IsZero() {
		return fmt.Errorf(
			"replace lessons for group %d: schedule update time is required",
			groupID,
		)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin replacing lessons for group %d: %w",
			groupID,
			err,
		)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var groupExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM groups
			WHERE id = ?
		)
	`, groupID).Scan(&groupExists)
	if err != nil {
		return fmt.Errorf("check group %d: %w", groupID, err)
	}

	if !groupExists {
		return fmt.Errorf("replace lessons for group %d: %w", groupID, errGroupNotFound)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM lessons
		WHERE group_id = ?
	`, groupID)
	if err != nil {
		return fmt.Errorf("delete old lessons for group %d: %w", groupID, err)
	}

	for _, lesson := range lessons {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO lessons (
				group_id,
				week_number,
				day_number,
				lesson_number,
				subgroup,
				subject,
				lesson_type,
				teacher_name,
				room,
				start_minute,
				end_minute
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			groupID,
			lesson.WeekNumber,
			lesson.DayNumber,
			lesson.LessonNumber,
			lesson.Subgroup,
			lesson.Subject,
			lesson.LessonType,
			lesson.TeacherName,
			lesson.Room,
			lesson.StartMinute,
			lesson.EndMinute,
		)
		if err != nil {
			return fmt.Errorf(
				"insert lesson for group %d (week %d, day %d, lesson %d, subgroup %d): %w",
				groupID,
				lesson.WeekNumber,
				lesson.DayNumber,
				lesson.LessonNumber,
				lesson.Subgroup,
				err,
			)
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE groups
		SET schedule_updated_at = ?
		WHERE id = ?
	`,
		scheduleUpdatedAt.UTC().Format(time.RFC3339Nano),
		groupID,
	)
	if err != nil {
		return fmt.Errorf(
			"update schedule timestamp for group %d: %w",
			groupID,
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"commit lesson replacement for group %d: %w",
			groupID,
			err,
		)
	}

	return nil
}

func loadGroupLessons(
	ctx context.Context,
	db *sql.DB,
	groupID int64,
) ([]Lesson, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf(
			"begin loading lessons for group %d: %w",
			groupID,
			err,
		)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var groupExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM groups
			WHERE id = ?
		)
	`, groupID).Scan(&groupExists)
	if err != nil {
		return nil, fmt.Errorf("check group %d: %w", groupID, err)
	}

	if !groupExists {
		return nil, fmt.Errorf(
			"load lessons for group %d: %w",
			groupID,
			errGroupNotFound,
		)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			week_number,
			day_number,
			lesson_number,
			subgroup,
			subject,
			lesson_type,
			teacher_name,
			room,
			start_minute,
			end_minute
		FROM lessons
		WHERE group_id = ?
		ORDER BY
			week_number,
			day_number,
			lesson_number,
			subgroup
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("query lessons for group %d: %w", groupID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	lessons := make([]Lesson, 0)

	for rows.Next() {
		var lesson Lesson

		err := rows.Scan(
			&lesson.WeekNumber,
			&lesson.DayNumber,
			&lesson.LessonNumber,
			&lesson.Subgroup,
			&lesson.Subject,
			&lesson.LessonType,
			&lesson.TeacherName,
			&lesson.Room,
			&lesson.StartMinute,
			&lesson.EndMinute,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"scan lesson for group %d: %w",
				groupID,
				err,
			)
		}

		lessons = append(lessons, lesson)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate lessons for group %d: %w",
			groupID,
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"commit loading lessons for group %d: %w",
			groupID,
			err,
		)
	}

	return lessons, nil
}
