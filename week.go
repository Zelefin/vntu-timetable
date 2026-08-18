package main

import (
	"fmt"
	"time"
)

func calculateUniversityWeek(date time.Time, offset int) (int, error) {
	if offset != 0 && offset != 1 {
		return 0, fmt.Errorf("week offset must be 0 or 1, got %d", offset)
	}

	_, isoWeek := date.ISOWeek()

	return (isoWeek+offset)%2 + 1, nil
}

func scheduleDayNumber(date time.Time) int {
	return (int(date.Weekday()) + 6) % 7
}

func calculateWeekOffset(
	date time.Time,
	desiredUniversityWeek int,
) (int, error) {
	if desiredUniversityWeek != 1 && desiredUniversityWeek != 2 {
		return 0, fmt.Errorf(
			"university week must be 1 or 2, got %d",
			desiredUniversityWeek,
		)
	}

	_, isoWeek := date.ISOWeek()
	weekWithoutOffset := isoWeek%2 + 1

	if weekWithoutOffset == desiredUniversityWeek {
		return 0, nil
	}

	return 1, nil
}
