package utils

import (
	"testing"
	"time"
)

type Profile struct {
	CreateTime time.Time
}

type User struct {
	ID      int
	Name    string
	Profile Profile
}

func TestNoField(t *testing.T) {
	users := []User{
		{ID: 3, Name: "Charlie"},
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	err := SortStructSlice(users, "", true, SnakeToCamelMapper())
	if err != nil {
		t.Fatalf("sort error: %v", err)
	}

	if users[0].ID != 3 || users[1].ID != 1 || users[2].ID != 2 {
		t.Fatalf("unexpected sort result: %+v", users)
	}
}

func TestNestedSort(t *testing.T) {
	now := time.Now()
	users := []User{
		{Profile: Profile{CreateTime: now.Add(5 * time.Hour)}},
		{Profile: Profile{CreateTime: now.Add(1 * time.Hour)}},
		{Profile: Profile{CreateTime: now.Add(3 * time.Hour)}},
	}

	err := SortStructSlice(users, "profile.create_time", true, SnakeToCamelMapper())
	if err != nil {
		t.Fatalf("sort error: %v", err)
	}

	if !users[0].Profile.CreateTime.Before(users[1].Profile.CreateTime) {
		t.Fatalf("unexpected sort order")
	}
}

func TestWithStaticMapping(t *testing.T) {
	users := []User{
		{ID: 3, Name: "Charlie"},
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	mapping := map[string]string{
		"id": "ID",
	}

	err := SortStructSlice(users, "id", true, StaticMapper(mapping))
	if err != nil {
		t.Fatalf("sort error: %v", err)
	}

	if users[0].ID != 1 {
		t.Fatalf("unexpected sort result: %+v", users)
	}
}

func TestCombinedMapper(t *testing.T) {
	users := []User{
		{ID: 3, Name: "Charlie"},
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	mapping := map[string]string{
		"id": "ID",
	}
	mapper := CombinedMapper(mapping)

	err := SortStructSlice(users, "id", true, mapper)
	if err != nil {
		t.Fatalf("sort error: %v", err)
	}
	if users[0].ID != 1 {
		t.Fatalf("unexpected sort result: %+v", users)
	}
}
