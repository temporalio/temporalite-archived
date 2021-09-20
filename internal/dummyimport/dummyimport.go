// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package dummyimport

// The !cgo build tag excludes some dependencies by default.
//
// Anonymous import(s) here ensure that these dependencies are
// included in third party license checks, etc.
import (
	_ "modernc.org/sqlite"
)
