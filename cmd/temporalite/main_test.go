// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package main

import (
	"reflect"
	"testing"
)

func TestGetDynamicConfigValues(t *testing.T) {
	assertBadVal := func(v string) {
		if _, err := getDynamicConfigValues([]string{v}); err == nil {
			t.Fatalf("expected error for %v", v)
		}
	}
	type v map[string][]interface{}
	assertGoodVals := func(expected v, in ...string) {
		actualVals, err := getDynamicConfigValues(in)
		if err != nil {
			t.Fatal(err)
		}
		actual := make(v, len(actualVals))
		for k, vals := range actualVals {
			for _, val := range vals {
				actual[string(k)] = append(actual[string(k)], val.Value)
			}
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("not equal, expected - actual: %v - %v", expected, actual)
		}
	}

	assertBadVal("foo")
	assertBadVal("foo=")
	assertBadVal("foo=bar")
	assertBadVal("foo=123a")

	assertGoodVals(v{"foo": {123.0}}, "foo=123")
	assertGoodVals(
		v{"foo": {123.0, []interface{}{"123", false}}, "bar": {"baz"}, "qux": {true}},
		"foo=123", `bar="baz"`, "qux=true", `foo=["123", false]`,
	)
}
