// MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2021 Datadog, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
