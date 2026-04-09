package repo

import (
	"reflect"
	"testing"
)

func TestBuildRandomCourseWindows(t *testing.T) {
	tests := []struct {
		name  string
		total int
		limit int
		start int
		want  []randomCourseWindow
	}{
		{
			name:  "invalid limit",
			total: 10,
			limit: 0,
			start: 3,
			want:  nil,
		},
		{
			name:  "empty dataset",
			total: 0,
			limit: 4,
			start: 1,
			want:  nil,
		},
		{
			name:  "single window without wrap",
			total: 10,
			limit: 4,
			start: 3,
			want: []randomCourseWindow{
				{Offset: 3, Limit: 4},
			},
		},
		{
			name:  "wraps to the beginning",
			total: 10,
			limit: 4,
			start: 8,
			want: []randomCourseWindow{
				{Offset: 8, Limit: 2},
				{Offset: 0, Limit: 2},
			},
		},
		{
			name:  "limit larger than total",
			total: 3,
			limit: 10,
			start: 1,
			want: []randomCourseWindow{
				{Offset: 1, Limit: 2},
				{Offset: 0, Limit: 1},
			},
		},
		{
			name:  "negative start is normalized",
			total: 5,
			limit: 2,
			start: -1,
			want: []randomCourseWindow{
				{Offset: 4, Limit: 1},
				{Offset: 0, Limit: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRandomCourseWindows(tt.total, tt.limit, tt.start)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildRandomCourseWindows(%d, %d, %d) = %#v, want %#v", tt.total, tt.limit, tt.start, got, tt.want)
			}
		})
	}
}
