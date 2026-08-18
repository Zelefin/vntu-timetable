package main

import (
	"testing"
	"time"
)

func TestCalculateUniversityWeek(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		offset   int
		expected int
	}{
		{
			name:     "academic year starts on even ISO week",
			date:     time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC),
			offset:   0,
			expected: 1,
		},
		{
			name:     "following odd ISO week",
			date:     time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
			offset:   0,
			expected: 2,
		},
		{
			name:     "offset flips even ISO week",
			date:     time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC),
			offset:   1,
			expected: 2,
		},
		{
			name:     "offset flips odd ISO week",
			date:     time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
			offset:   1,
			expected: 1,
		},
		{
			name:     "ISO week 53",
			date:     time.Date(2026, time.December, 28, 12, 0, 0, 0, time.UTC),
			offset:   0,
			expected: 2,
		},
		{
			name:     "ISO week 1 after ISO week 53",
			date:     time.Date(2027, time.January, 4, 12, 0, 0, 0, time.UTC),
			offset:   0,
			expected: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := calculateUniversityWeek(test.date, test.offset)
			if err != nil {
				t.Fatalf("calculateUniversityWeek() error = %v", err)
			}

			if got != test.expected {
				t.Errorf(
					"calculateUniversityWeek() = %d, expected %d",
					got,
					test.expected,
				)
			}
		})
	}
}

func TestCalculateUniversityWeekRejectsInvalidOffset(t *testing.T) {
	date := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

	for _, offset := range []int{-1, 2} {
		_, err := calculateUniversityWeek(date, offset)
		if err == nil {
			t.Errorf(
				"calculateUniversityWeek() with offset %d returned no error",
				offset,
			)
		}
	}
}

func TestScheduleDayNumber(t *testing.T) {
	monday := time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC)

	for expected := 0; expected <= 6; expected++ {
		date := monday.AddDate(0, 0, expected)

		t.Run(date.Weekday().String(), func(t *testing.T) {
			got := scheduleDayNumber(date)

			if got != expected {
				t.Errorf(
					"scheduleDayNumber() = %d, expected %d",
					got,
					expected,
				)
			}
		})
	}
}

func TestCalculateWeekOffset(t *testing.T) {
	tests := []struct {
		name                  string
		date                  time.Time
		desiredUniversityWeek int
		expectedOffset        int
	}{
		{
			name:                  "even ISO week is university week 1",
			date:                  time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC),
			desiredUniversityWeek: 1,
			expectedOffset:        0,
		},
		{
			name:                  "even ISO week is university week 2",
			date:                  time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC),
			desiredUniversityWeek: 2,
			expectedOffset:        1,
		},
		{
			name:                  "odd ISO week is university week 1",
			date:                  time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
			desiredUniversityWeek: 1,
			expectedOffset:        1,
		},
		{
			name:                  "odd ISO week is university week 2",
			date:                  time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
			desiredUniversityWeek: 2,
			expectedOffset:        0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := calculateWeekOffset(
				test.date,
				test.desiredUniversityWeek,
			)
			if err != nil {
				t.Fatalf("calculateWeekOffset() error = %v", err)
			}

			if got != test.expectedOffset {
				t.Errorf(
					"calculateWeekOffset() = %d, expected %d",
					got,
					test.expectedOffset,
				)
			}
		})
	}
}

func TestCalculateWeekOffsetRejectsInvalidUniversityWeek(t *testing.T) {
	date := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)

	for _, desiredUniversityWeek := range []int{0, 3} {
		_, err := calculateWeekOffset(date, desiredUniversityWeek)
		if err == nil {
			t.Errorf(
				"calculateWeekOffset() with university week %d returned no error",
				desiredUniversityWeek,
			)
		}
	}
}
