package main

import (
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	scheduleTestFacultyID int64 = 210
	scheduleTestGroupID   int64 = 10840
)

func openScheduleTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	ctx := t.Context()
	databasePath := filepath.Join(t.TempDir(), "test.db")

	db, err := openDatabase(ctx, databasePath)
	if err != nil {
		t.Fatalf("openDatabase() error = %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})

	if err := migrateDatabase(ctx, db); err != nil {
		t.Fatalf("migrateDatabase() error = %v", err)
	}

	_, err = db.ExecContext(ctx, `
  		INSERT INTO faculties (id, name)
  		VALUES (?, ?)
  	`, scheduleTestFacultyID, "Test Faculty")
	if err != nil {
		t.Fatalf("insert faculty: %v", err)
	}

	_, err = db.ExecContext(ctx, `
  		INSERT INTO groups (id, faculty_id, name)
  		VALUES (?, ?, ?)
  	`, scheduleTestGroupID, scheduleTestFacultyID, "Test Group")
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}

	return db
}

func TestReplaceGroupLessons(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	lesson := Lesson{
		WeekNumber:   1,
		DayNumber:    0,
		LessonNumber: 1,
		Subgroup:     0,
		Subject:      "Test Subject",
		LessonType:   "Lecture",
		TeacherName:  "Test Teacher",
		Room:         "101",
		StartMinute:  8 * 60,
		EndMinute:    9*60 + 20,
	}

	kyivSummerTime := time.FixedZone("UTC+3", 3*60*60)
	updatedAt := time.Date(
		2026,
		time.September,
		1,
		7,
		15,
		0,
		0,
		kyivSummerTime,
	)

	err := replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		[]Lesson{lesson},
		updatedAt,
	)
	if err != nil {
		t.Fatalf("replaceGroupLessons() error = %v", err)
	}

	var storedLesson Lesson
	err = db.QueryRowContext(ctx, `
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
	`, scheduleTestGroupID).Scan(
		&storedLesson.WeekNumber,
		&storedLesson.DayNumber,
		&storedLesson.LessonNumber,
		&storedLesson.Subgroup,
		&storedLesson.Subject,
		&storedLesson.LessonType,
		&storedLesson.TeacherName,
		&storedLesson.Room,
		&storedLesson.StartMinute,
		&storedLesson.EndMinute,
	)
	if err != nil {
		t.Fatalf("read stored lesson: %v", err)
	}

	if storedLesson != lesson {
		t.Errorf("stored lesson = %+v, expected %+v", storedLesson, lesson)
	}

	var scheduleUpdatedAt string
	err = db.QueryRowContext(ctx, `
		SELECT schedule_updated_at
		FROM groups
		WHERE id = ?
	`, scheduleTestGroupID).Scan(&scheduleUpdatedAt)
	if err != nil {
		t.Fatalf("read schedule_updated_at: %v", err)
	}

	const expectedUpdatedAt = "2026-09-01T04:15:00Z"
	if scheduleUpdatedAt != expectedUpdatedAt {
		t.Errorf(
			"schedule_updated_at = %q, expected %q",
			scheduleUpdatedAt,
			expectedUpdatedAt,
		)
	}
}

func TestReplaceGroupLessonsReplacesExistingLessons(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	oldLessons := []Lesson{
		{
			WeekNumber:   1,
			DayNumber:    0,
			LessonNumber: 1,
			Subgroup:     0,
			Subject:      "Old Subject One",
			LessonType:   "Lecture",
			TeacherName:  "Old Teacher",
			Room:         "101",
			StartMinute:  8 * 60,
			EndMinute:    9*60 + 20,
		},
		{
			WeekNumber:   1,
			DayNumber:    1,
			LessonNumber: 2,
			Subgroup:     0,
			Subject:      "Old Subject Two",
			LessonType:   "Practice",
			TeacherName:  "Old Teacher",
			Room:         "102",
			StartMinute:  9*60 + 30,
			EndMinute:    10*60 + 50,
		},
	}

	firstUpdatedAt := time.Date(2026, time.September, 1, 7, 0, 0, 0, time.UTC)
	err := replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		oldLessons,
		firstUpdatedAt,
	)
	if err != nil {
		t.Fatalf("first replaceGroupLessons() error = %v", err)
	}

	replacement := Lesson{
		WeekNumber:   2,
		DayNumber:    3,
		LessonNumber: 3,
		Subgroup:     1,
		Subject:      "Replacement Subject",
		LessonType:   "Laboratory",
		TeacherName:  "New Teacher",
		Room:         "201",
		StartMinute:  11 * 60,
		EndMinute:    12*60 + 20,
	}

	secondUpdatedAt := firstUpdatedAt.Add(time.Hour)
	err = replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		[]Lesson{replacement},
		secondUpdatedAt,
	)
	if err != nil {
		t.Fatalf("second replaceGroupLessons() error = %v", err)
	}

	var lessonCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM lessons
		WHERE group_id = ?
	`, scheduleTestGroupID).Scan(&lessonCount)
	if err != nil {
		t.Fatalf("count stored lessons: %v", err)
	}

	if lessonCount != 1 {
		t.Errorf("stored lesson count = %d, expected 1", lessonCount)
	}

	var storedSubject string
	err = db.QueryRowContext(ctx, `
		SELECT subject
		FROM lessons
		WHERE group_id = ?
	`, scheduleTestGroupID).Scan(&storedSubject)
	if err != nil {
		t.Fatalf("read stored subject: %v", err)
	}

	if storedSubject != replacement.Subject {
		t.Errorf(
			"stored subject = %q, expected %q",
			storedSubject,
			replacement.Subject,
		)
	}
}

func TestReplaceGroupLessonsAcceptsEmptySchedule(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	existingLesson := Lesson{
		WeekNumber:   1,
		DayNumber:    0,
		LessonNumber: 1,
		Subgroup:     0,
		Subject:      "Existing Subject",
		LessonType:   "Lecture",
		TeacherName:  "Teacher",
		Room:         "101",
		StartMinute:  8 * 60,
		EndMinute:    9*60 + 20,
	}

	firstUpdatedAt := time.Date(2026, time.September, 1, 7, 0, 0, 0, time.UTC)
	err := replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		[]Lesson{existingLesson},
		firstUpdatedAt,
	)
	if err != nil {
		t.Fatalf("initial replaceGroupLessons() error = %v", err)
	}

	emptyScheduleUpdatedAt := firstUpdatedAt.Add(time.Hour)
	err = replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		nil,
		emptyScheduleUpdatedAt,
	)
	if err != nil {
		t.Fatalf("empty replaceGroupLessons() error = %v", err)
	}

	var lessonCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM lessons
		WHERE group_id = ?
	`, scheduleTestGroupID).Scan(&lessonCount)
	if err != nil {
		t.Fatalf("count stored lessons: %v", err)
	}

	if lessonCount != 0 {
		t.Errorf("stored lesson count = %d, expected 0", lessonCount)
	}

	var scheduleUpdatedAt string
	err = db.QueryRowContext(ctx, `
		SELECT schedule_updated_at
		FROM groups
		WHERE id = ?
	`, scheduleTestGroupID).Scan(&scheduleUpdatedAt)
	if err != nil {
		t.Fatalf("read schedule_updated_at: %v", err)
	}

	expectedUpdatedAt := emptyScheduleUpdatedAt.Format(time.RFC3339Nano)
	if scheduleUpdatedAt != expectedUpdatedAt {
		t.Errorf(
			"schedule_updated_at = %q, expected %q",
			scheduleUpdatedAt,
			expectedUpdatedAt,
		)
	}
}

func TestReplaceGroupLessonsRollsBackInvalidReplacement(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	existingLesson := Lesson{
		WeekNumber:   1,
		DayNumber:    0,
		LessonNumber: 1,
		Subgroup:     0,
		Subject:      "Existing Subject",
		LessonType:   "Lecture",
		TeacherName:  "Existing Teacher",
		Room:         "101",
		StartMinute:  8 * 60,
		EndMinute:    9*60 + 20,
	}

	originalUpdatedAt := time.Date(
		2026,
		time.September,
		1,
		7,
		0,
		0,
		0,
		time.UTC,
	)

	err := replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		[]Lesson{existingLesson},
		originalUpdatedAt,
	)
	if err != nil {
		t.Fatalf("initial replaceGroupLessons() error = %v", err)
	}

	validReplacement := Lesson{
		WeekNumber:   2,
		DayNumber:    1,
		LessonNumber: 1,
		Subgroup:     0,
		Subject:      "Valid Replacement",
		LessonType:   "Practice",
		TeacherName:  "New Teacher",
		Room:         "201",
		StartMinute:  9 * 60,
		EndMinute:    10 * 60,
	}

	invalidReplacement := Lesson{
		WeekNumber:   1,
		DayNumber:    2,
		LessonNumber: 2,
		Subgroup:     0,
		Subject:      "Invalid Replacement",
		LessonType:   "Laboratory",
		TeacherName:  "New Teacher",
		Room:         "202",
		StartMinute:  11 * 60,
		EndMinute:    11 * 60,
	}

	failedUpdatedAt := originalUpdatedAt.Add(time.Hour)
	err = replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		[]Lesson{validReplacement, invalidReplacement},
		failedUpdatedAt,
	)
	if err == nil {
		t.Fatal("replaceGroupLessons() error = nil, expected error")
	}

	expectedErrorDetail := "week 1, day 2, lesson 2, subgroup 0"
	if !strings.Contains(err.Error(), expectedErrorDetail) {
		t.Errorf(
			"replaceGroupLessons() error = %q, expected it to contain %q",
			err,
			expectedErrorDetail,
		)
	}

	var lessonCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM lessons
		WHERE group_id = ?
	`, scheduleTestGroupID).Scan(&lessonCount)
	if err != nil {
		t.Fatalf("count stored lessons: %v", err)
	}

	if lessonCount != 1 {
		t.Errorf("stored lesson count = %d, expected 1", lessonCount)
	}

	var storedSubject string
	err = db.QueryRowContext(ctx, `
		SELECT subject
		FROM lessons
		WHERE group_id = ?
	`, scheduleTestGroupID).Scan(&storedSubject)
	if err != nil {
		t.Fatalf("read stored subject: %v", err)
	}

	if storedSubject != existingLesson.Subject {
		t.Errorf(
			"stored subject = %q, expected preserved subject %q",
			storedSubject,
			existingLesson.Subject,
		)
	}

	var scheduleUpdatedAt string
	err = db.QueryRowContext(ctx, `
		SELECT schedule_updated_at
		FROM groups
		WHERE id = ?
	`, scheduleTestGroupID).Scan(&scheduleUpdatedAt)
	if err != nil {
		t.Fatalf("read schedule_updated_at: %v", err)
	}

	expectedUpdatedAt := originalUpdatedAt.Format(time.RFC3339Nano)
	if scheduleUpdatedAt != expectedUpdatedAt {
		t.Errorf(
			"schedule_updated_at = %q, expected preserved value %q",
			scheduleUpdatedAt,
			expectedUpdatedAt,
		)
	}
}

func TestReplaceGroupLessonsRequiresUpdateTime(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	err := replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		nil,
		time.Time{},
	)
	if err == nil {
		t.Fatal("replaceGroupLessons() error = nil, expected error")
	}

	const expectedDetail = "schedule update time is required"
	if !strings.Contains(err.Error(), expectedDetail) {
		t.Errorf(
			"replaceGroupLessons() error = %q, expected it to contain %q",
			err,
			expectedDetail,
		)
	}
}

func TestReplaceGroupLessonsRejectsUnknownGroup(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	const unknownGroupID int64 = 999999

	err := replaceGroupLessons(
		ctx,
		db,
		unknownGroupID,
		nil,
		time.Date(2026, time.September, 1, 7, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("replaceGroupLessons() error = nil, expected error")
	}

	if !errors.Is(err, errGroupNotFound) {
		t.Errorf(
			"replaceGroupLessons() error = %q, expected errGroupNotFound",
			err,
		)
	}
}

func TestLoadGroupLessonsReturnsDeterministicOrder(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	newLesson := func(
		week int,
		day int,
		number int,
		subgroup int,
		subject string,
	) Lesson {
		return Lesson{
			WeekNumber:   week,
			DayNumber:    day,
			LessonNumber: number,
			Subgroup:     subgroup,
			Subject:      subject,
			LessonType:   "Test Type",
			TeacherName:  "Test Teacher",
			Room:         "101",
			StartMinute:  8 * 60,
			EndMinute:    9 * 60,
		}
	}

	groupWide := newLesson(1, 0, 1, 0, "Group Wide")
	subgroupOne := newLesson(1, 0, 1, 1, "Subgroup One")
	subgroupTwo := newLesson(1, 0, 1, 2, "Subgroup Two")
	laterLesson := newLesson(1, 0, 2, 0, "Later Lesson")
	laterDay := newLesson(1, 1, 1, 0, "Later Day")
	secondWeek := newLesson(2, 0, 1, 0, "Second Week")

	unsortedLessons := []Lesson{
		secondWeek,
		subgroupTwo,
		laterDay,
		laterLesson,
		subgroupOne,
		groupWide,
	}

	err := replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		unsortedLessons,
		time.Date(2026, time.September, 1, 7, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("replaceGroupLessons() error = %v", err)
	}

	got, err := loadGroupLessons(ctx, db, scheduleTestGroupID)
	if err != nil {
		t.Fatalf("loadGroupLessons() error = %v", err)
	}

	expected := []Lesson{
		groupWide,
		subgroupOne,
		subgroupTwo,
		laterLesson,
		laterDay,
		secondWeek,
	}

	if !slices.Equal(got, expected) {
		t.Errorf("loadGroupLessons() = %+v, expected %+v", got, expected)
	}
}

func TestLoadGroupLessonsReturnsEmptySlice(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	lessons, err := loadGroupLessons(ctx, db, scheduleTestGroupID)
	if err != nil {
		t.Fatalf("loadGroupLessons() error = %v", err)
	}

	if lessons == nil {
		t.Fatal("loadGroupLessons() returned nil, expected empty slice")
	}

	if len(lessons) != 0 {
		t.Errorf(
			"loadGroupLessons() returned %d lessons, expected 0",
			len(lessons),
		)
	}
}

func TestLoadGroupLessonsRejectsUnknownGroup(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	const unknownGroupID int64 = 999999

	_, err := loadGroupLessons(ctx, db, unknownGroupID)
	if err == nil {
		t.Fatal("loadGroupLessons() error = nil, expected error")
	}

	if !errors.Is(err, errGroupNotFound) {
		t.Errorf(
			"loadGroupLessons() error = %v, expected errGroupNotFound",
			err,
		)
	}
}

func TestReplaceGroupLessonsRejectsDuplicateSlot(t *testing.T) {
	ctx := t.Context()
	db := openScheduleTestDatabase(t)

	firstLesson := Lesson{
		WeekNumber:   1,
		DayNumber:    0,
		LessonNumber: 1,
		Subgroup:     1,
		Subject:      "First Subject",
		LessonType:   "Lecture",
		TeacherName:  "First Teacher",
		Room:         "101",
		StartMinute:  8 * 60,
		EndMinute:    9*60 + 20,
	}

	duplicateSlot := firstLesson
	duplicateSlot.Subject = "Conflicting Subject"
	duplicateSlot.TeacherName = "Another Teacher"
	duplicateSlot.Room = "102"

	err := replaceGroupLessons(
		ctx,
		db,
		scheduleTestGroupID,
		[]Lesson{firstLesson, duplicateSlot},
		time.Date(2026, time.September, 1, 7, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("replaceGroupLessons() error = nil, expected error")
	}

	const expectedDetail = "week 1, day 0, lesson 1, subgroup 1"
	if !strings.Contains(err.Error(), expectedDetail) {
		t.Errorf(
			"replaceGroupLessons() error = %q, expected it to contain %q",
			err,
			expectedDetail,
		)
	}

	storedLessons, err := loadGroupLessons(ctx, db, scheduleTestGroupID)
	if err != nil {
		t.Fatalf("loadGroupLessons() error = %v", err)
	}

	if len(storedLessons) != 0 {
		t.Errorf(
			"stored lesson count = %d, expected 0 after rollback",
			len(storedLessons),
		)
	}
}
