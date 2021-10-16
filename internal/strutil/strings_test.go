package strutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRemoveExtraSpaces(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "test_not_modifying_0",
			source: "Hello world!",
			want:   "Hello world!",
		},
		{
			name:   "test_not_extra_space_inner",
			source: "Hello  world!",
			want:   "Hello world!",
		},
		{
			name: "test_not_extra_space_inner_tab",
			source: "Hello        	world!",
			want: "Hello world!",
		},
		{
			name:   "test_not_extra_space_inner_outer",
			source: "   Hello        world!   ",
			want:   "Hello world!",
		},
		{
			name: "test_not_extra_space_inner_outer_tab_0",
			source: "   Hello        	world!   ",
			want: "Hello world!",
		},
		{
			name: "test_not_extra_space_inner_outer_tab_1",
			source: "   	Hello        	w.  o.  r.  l.  d!   ",
			want: "Hello w. o. r. l. d!",
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := RemoveExtraSpaces(test.source)
			if got != test.want {
				diff := cmp.Diff(test.want, got)
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestCamelCase(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "test_camel_case_0",
			source: "Hello world!",
			want:   "HelloWorld!",
		},
		{
			name:   "test_camel_case_1",
			source: "HiHello world!",
			want:   "HihelloWorld!",
		},
		{
			name:   "test_camel_case_2",
			source: "Hi my name is robotomize",
			want:   "HiMyNameIsRobotomize",
		},
		{
			name:   "test_extra_space_camel_case",
			source: "Hello   world!",
			want:   "HelloWorld!",
		},
		{
			name: "test_tab_camel_case",
			source: "Hello  	world!",
			want: "Hello	World!",
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := CamelCase(test.source)
			if got != test.want {
				diff := cmp.Diff(test.want, got)
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestRemoveContentIntoBrackets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "test_remove_into_brackets_0",
			source: "(Hello world)",
			want:   "",
		},
		{
			name:   "test_remove_into_brackets_1",
			source: "[Hello world]",
			want:   "",
		},
		{
			name:   "test_remove_into_brackets_2",
			source: "[Hello world](Hello world)",
			want:   "",
		},
		{
			name:   "test_remove_into_brackets_3",
			source: "Hello [Hello world]world!(Hello world)",
			want:   "Hello world!",
		},
		{
			name:   "test_remove_into_brackets_3",
			source: "Hello [Hello (world)]world!(Hello [world])",
			want:   "Hello world!",
		},
		{
			name:   "test_remove_into_brackets_4",
			source: "Hello [Hello ((world))]world!([Hello] [world])",
			want:   "Hello world!",
		},

		// The case does not work

		//{
		//	name:   "test_remove_into_brackets_5",
		//	source: "Hello [world [hello]]",
		//	want:   "Hello world!",
		//},
	}
	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := RemoveContentIntoBrackets(test.source)
			if got != test.want {
				diff := cmp.Diff(test.want, got)
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestRemoveNonAlphaNum(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "test_remove_non_alpha_0",
			source: "Hello world!",
			want:   "Hello world",
		},
		{
			name:   "test_remove_non_alpha_1",
			source: "[Hello] (world)!",
			want:   "Hello world",
		},
		{
			name:   "test_remove_non_alpha_2",
			source: "(((((((([Hello] (world)))))))))!",
			want:   "Hello world",
		},
		{
			name:   "test_remove_non_alpha_3",
			source: `!#$%*@?<>'\/~(){}[]&^Hello world`,
			want:   "Hello world",
		},
		{
			name:   "test_remove_non_alpha_4",
			source: `ПриветмирHello world`,
			want:   "Hello world",
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := RemoveNonAlphaNum(test.source)
			if got != test.want {
				diff := cmp.Diff(test.want, got)
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
