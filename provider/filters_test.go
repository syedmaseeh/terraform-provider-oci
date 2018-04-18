package provider

import (
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	oci_core "github.com/oracle/oci-go-sdk/core"
)

// Not supplying filters should not restrict results
func TestApplyFilters_passThrough(t *testing.T) {
	items := []map[string]interface{}{
		{},
		{},
		{},
	}

	res := ApplyFilters(nil, items)
	if len(res) != 3 {
		t.Errorf("Expected 3 results, got %d", len(res))
	}
}

// Filtering against a nonexistent property should throw no errors and return no results
func TestApplyFilters_nonExistentProperty(t *testing.T) {
	items := []map[string]interface{}{
		{"letter": "a"},
	}

	filters := &schema.Set{F: func(interface{}) int { return 1 }}
	filters.Add(map[string]interface{}{
		"name":   "number",
		"values": []interface{}{"1"},
	})

	res := ApplyFilters(filters, items)
	if len(res) > 0 {
		t.Errorf("Expected 0 results, got %d", len(res))
	}
}

// Filtering against an empty resource set should not throw errors
func TestApplyFilters_noResources(t *testing.T) {
	items := []map[string]interface{}{}

	filters := &schema.Set{F: func(interface{}) int { return 1 }}
	filters.Add(map[string]interface{}{
		"name":   "number",
		"values": []interface{}{"1"},
	})

	res := ApplyFilters(filters, items)
	if len(res) != 0 {
		t.Errorf("Expected 0 results, got %d", len(res))
	}
}

func TestApplyFilters_basic(t *testing.T) {
	items := []map[string]interface{}{
		{"letter": "a"},
		{"letter": "b"},
		{"letter": "c"},
	}

	filters := &schema.Set{F: func(interface{}) int { return 1 }}
	filters.Add(map[string]interface{}{
		"name":   "letter",
		"values": []interface{}{"b"},
	})

	res := ApplyFilters(filters, items)
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got %d", len(res))
	}
}

func TestApplyFilters_duplicates(t *testing.T) {
	items := []map[string]interface{}{
		{"letter": "a"},
		{"letter": "a"},
		{"letter": "c"},
	}

	filters := &schema.Set{F: func(v interface{}) int {
		return schema.HashString(v.(map[string]interface{})["name"])
	}}
	filters.Add(map[string]interface{}{
		"name":   "letter",
		"values": []interface{}{"a"},
	})

	res := ApplyFilters(filters, items)
	if len(res) != 2 {
		t.Errorf("Expected 2 results, got %d", len(res))
	}
}

func TestApplyFilters_OR(t *testing.T) {
	items := []map[string]interface{}{
		{"letter": "a"},
		{"letter": "b"},
		{"letter": "c"},
	}

	filters := &schema.Set{
		F: func(v interface{}) int {
			elems := v.(map[string]interface{})["values"].([]interface{})
			res := make([]string, len(elems))
			for i, v := range elems {
				res[i] = v.(string)
			}
			return schema.HashString(strings.Join(res, ""))
		},
	}
	filters.Add(map[string]interface{}{
		"name":   "letter",
		"values": []interface{}{"a", "b"},
	})

	res := ApplyFilters(filters, items)
	if len(res) != 2 {
		t.Errorf("Expected 2 results, got %d", len(res))
	}
}

func TestApplyFilters_cascadeAND(t *testing.T) {
	items := []map[string]interface{}{
		{"letter": "a"},
		{"letter": "b"},
		{"letter": "c"},
	}

	filters := &schema.Set{
		F: func(v interface{}) int {
			elems := v.(map[string]interface{})["values"].([]interface{})
			res := make([]string, len(elems))
			for i, v := range elems {
				res[i] = v.(string)
			}
			return schema.HashString(strings.Join(res, ""))
		},
	}
	filters.Add(map[string]interface{}{
		"name":   "letter",
		"values": []interface{}{"a", "b"},
	})
	filters.Add(map[string]interface{}{
		"name":   "letter",
		"values": []interface{}{"c"},
	})

	res := ApplyFilters(filters, items)
	if len(res) != 0 {
		t.Errorf("Expected 0 results, got %d", len(res))
	}
}

func TestApplyFilters_regex(t *testing.T) {
	items := []map[string]interface{}{
		{"string": "xblx:PHX-AD-1"},
		{"string": "xblx:PHX-AD-2"},
		{"string": "xblx:PHX-AD-3"},
	}

	filters := &schema.Set{F: func(v interface{}) int {
		return schema.HashString(v.(map[string]interface{})["name"])
	}}
	filters.Add(map[string]interface{}{
		"name":   "string",
		"values": []interface{}{"\\w*:PHX-AD-2"},
		"regex":  true,
	})

	res := ApplyFilters(filters, items)
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got %d", len(res))
	}
}

// Filters should test against an array of strings
func TestApplyFilters_arrayOfStrings(t *testing.T) {
	items := []map[string]interface{}{
		{"letters": []string{"a"}},
		{"letters": []string{"b", "c"}},
		{"letters": []string{"c", "d", "e"}},
		{"letters": []string{"e", "f"}},
	}

	filters := &schema.Set{F: func(interface{}) int { return 1 }}
	filters.Add(map[string]interface{}{
		"name":   "letters",
		"values": []interface{}{"a", "c"},
	})

	res := ApplyFilters(filters, items)
	if len(res) != 3 {
		t.Errorf("Expected 3 result, got %d", len(res))
	}

	filters = &schema.Set{F: func(interface{}) int { return 1 }}
	filters.Add(map[string]interface{}{
		"name":   "letters",
		"values": []interface{}{"a", "f"},
	})

	res = ApplyFilters(filters, items)
	if len(res) != 2 {
		t.Errorf("Expected 2 result, got %d", len(res))
	}
}

type CustomStringTypeA string
type CustomStringTypeB CustomStringTypeA
type CustomEnumType oci_core.VcnLifecycleStateEnum

func TestApplyFilters_underlyingStringTypes(t *testing.T) {
	items := []map[string]interface{}{
		{
			"letters": []CustomStringTypeA{"a"},
			"number":  CustomStringTypeB("1"),
			"state":   oci_core.SecurityListLifecycleStateAvailable,
			"custom":  CustomEnumType(oci_core.VcnLifecycleStateTerminated),
		},
		{
			"letters": []CustomStringTypeA{"a"},
			"number":  CustomStringTypeB("1"),
			"state":   oci_core.SecurityListLifecycleStateProvisioning,
			"custom":  CustomEnumType(oci_core.VcnLifecycleStateTerminating),
		},
		{
			"letters": []CustomStringTypeA{"b", "c"},
			"number":  CustomStringTypeB("2"),
			"state":   oci_core.SecurityListLifecycleStateTerminating,
			"custom":  CustomEnumType(oci_core.VcnLifecycleStateProvisioning),
		},
		{
			"letters": []CustomStringTypeA{"c", "d", "e"},
			"number":  CustomStringTypeB("3"),
			"state":   oci_core.SecurityListLifecycleStateTerminated,
			"custom":  CustomEnumType(oci_core.VcnLifecycleStateAvailable),
		},
		{
			"letters": []CustomStringTypeA{"e", "f"},
			"number":  CustomStringTypeB("5"),
			"state":   oci_core.SecurityListLifecycleStateAvailable,
			"custom":  CustomEnumType(oci_core.VcnLifecycleStateTerminated),
		},
	}

	filters := &schema.Set{
		F: func(v interface{}) int {
			return schema.HashString(v.(map[string]interface{})["name"])
		},
	}
	filters.Add(map[string]interface{}{
		"name":   "letters",
		"values": []interface{}{"a", "c"},
	})

	res := ApplyFilters(filters, items)
	if len(res) != 4 {
		t.Errorf("Expected 4 result, got %d", len(res))
	}

	filters1 := &schema.Set{
		F: func(v interface{}) int {
			return schema.HashString(v.(map[string]interface{})["name"])
		},
	}
	filters1.Add(map[string]interface{}{
		"name":   "letters",
		"values": []interface{}{"a", "b", "e"},
	})
	filters1.Add(map[string]interface{}{
		"name":   "number",
		"values": []interface{}{"1", "notANumber"},
	})

	res = ApplyFilters(filters1, items)
	if len(res) != 2 {
		t.Errorf("Expected 2 result, got %d", len(res))
	}

	filters2 := &schema.Set{
		F: func(v interface{}) int {
			return schema.HashString(v.(map[string]interface{})["name"])
		},
	}
	filters2.Add(map[string]interface{}{
		"name":   "letters",
		"values": []interface{}{"a", "b", "e"},
	})
	filters2.Add(map[string]interface{}{
		"name":   "number",
		"values": []interface{}{"1", "2", "3", "5"},
	})
	filters2.Add(map[string]interface{}{
		"name":   "state",
		"values": []interface{}{string(oci_core.SecurityListLifecycleStateAvailable), string(oci_core.SecurityListLifecycleStateTerminating)},
	})
	filters2.Add(map[string]interface{}{
		"name":   "custom",
		"values": []interface{}{string(oci_core.VcnLifecycleStateProvisioning)},
	})

	res = ApplyFilters(filters2, items)
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got %d", len(res))
	}
}

// Test fields that aren't supported: list of non-strings or structured objects
func TestApplyFilters_unsupportedTypes(t *testing.T) {
	items := []map[string]interface{}{
		{
			"nums": []int{1, 2, 3},
		},
		{
			"nums": []int{3, 4, 5},
		},
		{
			"nums": []int{5, 6, 7},
		},
	}

	filters := &schema.Set{
		F: func(v interface{}) int {
			return schema.HashString(v.(map[string]interface{})["name"])
		},
	}

	intArrayFilter := map[string]interface{}{
		"name":   "nums",
		"values": []interface{}{"1", "3", "5"},
	}
	filters.Add(intArrayFilter)

	res := ApplyFilters(filters, items)
	if len(res) != 0 {
		t.Errorf("Expected 0 result, got %d", len(res))
	}
}

func TestApplyFilters_booleanTypes(t *testing.T) {
	items := []map[string]interface{}{
		{
			"enabled": true,
		},
		{
			"enabled": "true",
		},
		{
			"enabled": "1",
		},
		{
			"enabled": false,
		},
		{
			"enabled": "false",
		},
		{
			"enabled": "0",
		},
	}

	filters := &schema.Set{
		F: func(v interface{}) int {
			return schema.HashString(v.(map[string]interface{})["name"])
		},
	}

	truthyBooleanFilter := map[string]interface{}{
		"name":   "enabled",
		"values": []interface{}{"true", "1"}, // while we can pass an actual boolean true here in the test, terraform
		// doesnt, so keep coercion logic simple in filters.go
	}
	filters.Add(truthyBooleanFilter)

	res := ApplyFilters(filters, items)

	for _, i := range res {
		switch enabled := i["enabled"].(type) {
		case bool:
			if !enabled {
				t.Errorf("Expected a truthy value, got %t", enabled)
			}
		case string:
			enabledBool, _ := strconv.ParseBool(enabled)
			if !enabledBool {
				t.Errorf("Expected a truthy value, got %s", enabled)
			}
		}
	}

	if len(res) != 3 {
		t.Errorf("Expected 3 results, got %d", len(res))
	}
	filters.Remove(truthyBooleanFilter)

	falsyBooleanFilter := map[string]interface{}{
		"name":   "enabled",
		"values": []interface{}{"false", "0"},
	}
	filters.Add(falsyBooleanFilter)

	res = ApplyFilters(filters, items)

	for _, i := range res {
		switch enabled := i["enabled"].(type) {
		case bool:
			if enabled {
				t.Errorf("Expected a falsy value, got %t", enabled)
			}
		case string:
			enabledBool, _ := strconv.ParseBool(enabled)
			if enabledBool {
				t.Errorf("Expected a falsy value, got %s", enabled)
			}
		}
	}

	if len(res) != 3 {
		t.Errorf("Expected 3 results, got %d", len(res))
	}
	filters.Remove(falsyBooleanFilter)
}

func TestApplyFilters_numberTypes(t *testing.T) {
	items := []map[string]interface{}{
		{
			"integer": 1,
			"float":   1.1,
		},
		{
			"integer": 2,
			"float":   2.2,
		},
		{
			"integer": 3,
			"float":   3.3,
		},
	}

	filters := &schema.Set{
		F: func(v interface{}) int {
			return schema.HashString(v.(map[string]interface{})["name"])
		},
	}

	// int filter with single target value
	intFilter := map[string]interface{}{
		"name":   "integer",
		"values": []interface{}{"2"},
	}
	filters.Add(intFilter)

	res := ApplyFilters(filters, items)
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got %d", len(res))
	}
	filters.Remove(intFilter)

	// test filter with multiple target value
	intsFilter := map[string]interface{}{
		"name":   "integer",
		"values": []interface{}{"1", "3"},
	}
	filters.Add(intsFilter)

	res = ApplyFilters(filters, items)
	if len(res) != 2 {
		t.Errorf("Expected 2 results, got %d", len(res))
	}
	filters.Remove(intsFilter)

	// test float filter
	floatFilter := map[string]interface{}{
		"name":   "float",
		"values": []interface{}{"1.1", "3.3"},
	}
	filters.Add(floatFilter)

	res = ApplyFilters(filters, items)
	if len(res) != 2 {
		t.Errorf("Expected 2 results, got %d", len(res))
	}
	filters.Remove(floatFilter)
}

func TestApplyFilters_multiProperty(t *testing.T) {
	items := []map[string]interface{}{
		{
			"letter": "a",
			"number": "1",
			"symbol": "!",
		},
		{
			"letter": "b",
			"number": "2",
			"symbol": "@",
		},
		{
			"letter": "c",
			"number": "3",
			"symbol": "#",
		},
		{
			"letter": "d",
			"number": "4",
			"symbol": "$",
		},
	}

	filters := &schema.Set{
		F: func(v interface{}) int {
			return schema.HashString(v.(map[string]interface{})["name"])
		},
	}
	filters.Add(map[string]interface{}{
		"name":   "letter",
		"values": []interface{}{"a", "b", "c"},
	})
	filters.Add(map[string]interface{}{
		"name":   "number",
		"values": []interface{}{"2", "3", "4"},
	})
	filters.Add(map[string]interface{}{
		"name":   "symbol",
		"values": []interface{}{"#", "$"},
	})

	res := ApplyFilters(filters, items)
	if len(res) != 1 {
		t.Errorf("Expected 1 result, got %d", len(res))
	}
}
