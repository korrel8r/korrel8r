// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mock_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_LoadFile(t *testing.T) {
	d := mock.NewDomain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)
	require.NoError(t, s.LoadFile("testdata/test_store.yaml"))

	q := mock.NewQuery(c, "query1")
	r := &mock.Result{}
	require.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []any{"x", "y"}, r.List())

	r = &mock.Result{}
	require.NoError(t, s.Get(context.Background(), mock.NewQuery(c, "query2"), nil, r))
	assert.Equal(t, []any{"a", "b", "c"}, r.List())
}

func TestStore_NewQuery(t *testing.T) {
	d := mock.NewDomain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)

	q1 := s.NewQuery(c, 1, 2)
	q2 := s.NewQuery(c, 3, 4)
	r := &mock.Result{}
	assert.NoError(t, s.Get(context.Background(), q1, nil, r))
	assert.Equal(t, []korrel8r.Object{1, 2}, r.List())
	r = &mock.Result{}
	assert.NoError(t, s.Get(context.Background(), q2, nil, r))
	assert.Equal(t, []korrel8r.Object{3, 4}, r.List())

	r = &mock.Result{}
	q3 := mock.NewQuery(c, "foo", 1, 2, 3)
	assert.NoError(t, s.Get(context.Background(), q3, nil, r))
	assert.Equal(t, []korrel8r.Object{1, 2, 3}, r.List())
}

func TestStore_Get(t *testing.T) {
	d := mock.NewDomain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)
	q := mock.NewQuery(c, "query")
	s.AddQuery(q, []korrel8r.Object{"a", "b"})
	r := &mock.Result{}
	require.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []korrel8r.Object{"a", "b"}, r.List())
}

func TestFileStore(t *testing.T) {
	d := mock.NewDomain("foo")
	c := d.Class("x")
	s := mock.NewStore(d)
	s.AddLookup(mock.QueryDir("testdata/_filestore").Get)
	q := mock.NewQuery(c, "query1")
	r := &mock.Result{}
	require.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []any{"x", "y"}, r.List())

	r = &mock.Result{}
	require.NoError(t, s.Get(context.Background(), mock.NewQuery(c, "query2"), nil, r))
	assert.Equal(t, []any{"a", "b", "c"}, r.List())
}

func TestNewQueryError(t *testing.T) {
	d := mock.NewDomain("foo")
	s := mock.NewStore(d)
	q := mock.NewQueryError(d.Class("x"), "badQuery", errors.New("did not work"))
	err := s.Get(context.Background(), q, nil, &mock.Result{})
	assert.ErrorContains(t, err, "did not work")
}

// Domain tests
func TestNewDomain(t *testing.T) {
	d := mock.NewDomain("testdomain", "class1", "class2", "class3")

	assert.Equal(t, "testdomain", d.Name())
	assert.Equal(t, "testdomain", d.String())
	assert.Equal(t, "Mock domain.", d.Description())

	classes := d.Classes()
	assert.Len(t, classes, 3)

	assert.NotNil(t, d.Class("class1"))
	assert.NotNil(t, d.Class("class2"))
	assert.NotNil(t, d.Class("class3"))
	assert.Nil(t, d.Class("nonexistent"))
}

func TestDomain_EmptyClasses(t *testing.T) {
	d := mock.NewDomain("testdomain")

	assert.Empty(t, d.Classes())
	assert.NotNil(t, d.Class("anyclass"))
}

func TestDomain_Store(t *testing.T) {
	d := mock.NewDomain("testdomain")
	store, err := d.Store(nil)
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.Equal(t, d, store.Domain())
}

func TestDomain_Query(t *testing.T) {
	d := mock.NewDomain("testdomain", "class1")

	q, err := d.Query("testdomain:class1:selector")
	require.NoError(t, err)
	assert.Equal(t, "class1", q.Class().Name())
	assert.Equal(t, "selector", q.Data())

	_, err = d.Query("wrongdomain:class1:selector")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wrong query domain")

	_, err = d.Query("testdomain:nonexistent:selector")
	assert.Error(t, err)
}

// Class tests
func TestClass_Methods(t *testing.T) {
	d := mock.NewDomain("testdomain")
	c := d.Class("testclass")

	assert.Equal(t, "testclass", c.Name())
	assert.Equal(t, d, c.Domain())
	assert.Equal(t, "testdomain:testclass", c.String())
	obj := map[string]any{"test": "value"}
	assert.Equal(t, "map[test:value]", korrel8r.GetID(c, obj))
}

func TestClass_Unmarshal(t *testing.T) {
	d := mock.NewDomain("testdomain")
	c := d.Class("testclass")

	data := `{"key": "value", "number": 42}`
	obj, err := c.Unmarshal([]byte(data))
	require.NoError(t, err)

	expected := map[string]any{"key": "value", "number": float64(42)}
	assert.Equal(t, expected, obj)

	_, err = c.Unmarshal([]byte("invalid json"))
	assert.Error(t, err)
}

// Rule tests
func TestNewRule_WithApplyFunc(t *testing.T) {
	d := mock.NewDomain("testdomain")
	start := []korrel8r.Class{d.Class("start")}
	goal := []korrel8r.Class{d.Class("goal")}

	applyFunc := func(obj korrel8r.Object) (korrel8r.Query, error) {
		return mock.NewQuery(goal[0], "applied"), nil
	}

	rule := mock.NewRule("test-rule", start, goal, applyFunc)

	assert.Equal(t, "test-rule", rule.Name())
	assert.Equal(t, "test-rule", rule.String())
	assert.Equal(t, start, rule.Start())
	assert.Equal(t, goal, rule.Goal())

	q, err := rule.Apply("test-object")
	require.NoError(t, err)
	assert.Equal(t, "applied", q.Data())
}

func TestNewRule_WithQuery(t *testing.T) {
	d := mock.NewDomain("testdomain")
	start := []korrel8r.Class{d.Class("start")}
	goal := []korrel8r.Class{d.Class("goal")}
	query := mock.NewQuery(goal[0], "static-query")

	rule := mock.NewRule("test-rule", start, goal, query)

	q, err := rule.Apply("any-object")
	require.NoError(t, err)
	assert.Equal(t, "static-query", q.Data())
}

func TestNewRule_WithNil(t *testing.T) {
	d := mock.NewDomain("testdomain")
	start := []korrel8r.Class{d.Class("start")}
	goal := []korrel8r.Class{d.Class("goal")}

	rule := mock.NewRule("test-rule", start, goal, nil)

	_, err := rule.Apply("any-object")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock rule has no result")
}

func TestNewRule_WithInvalidType(t *testing.T) {
	d := mock.NewDomain("testdomain")
	start := []korrel8r.Class{d.Class("start")}
	goal := []korrel8r.Class{d.Class("goal")}

	assert.Panics(t, func() {
		mock.NewRule("test-rule", start, goal, "invalid")
	})
}

func TestRuleLess(t *testing.T) {
	d := mock.NewDomain("testdomain")

	rule1 := mock.NewRule("rule1", []korrel8r.Class{d.Class("a")}, []korrel8r.Class{d.Class("x")}, nil)
	rule2 := mock.NewRule("rule2", []korrel8r.Class{d.Class("a")}, []korrel8r.Class{d.Class("y")}, nil)
	rule3 := mock.NewRule("rule3", []korrel8r.Class{d.Class("b")}, []korrel8r.Class{d.Class("x")}, nil)

	assert.Less(t, mock.RuleLess(rule1, rule2), 0)
	assert.Greater(t, mock.RuleLess(rule2, rule1), 0)
	assert.Less(t, mock.RuleLess(rule1, rule3), 0)
	assert.Equal(t, 0, mock.RuleLess(rule1, rule1))
}

func TestSortRules(t *testing.T) {
	d := mock.NewDomain("testdomain")

	rule1 := mock.NewRule("rule1", []korrel8r.Class{d.Class("b")}, []korrel8r.Class{d.Class("y")}, nil)
	rule2 := mock.NewRule("rule2", []korrel8r.Class{d.Class("a")}, []korrel8r.Class{d.Class("z")}, nil)
	rule3 := mock.NewRule("rule3", []korrel8r.Class{d.Class("a")}, []korrel8r.Class{d.Class("x")}, nil)

	rules := []korrel8r.Rule{rule1, rule2, rule3}
	sorted := mock.SortRules(rules)

	assert.Equal(t, rule3, sorted[0])
	assert.Equal(t, rule2, sorted[1])
	assert.Equal(t, rule1, sorted[2])
}

// Query tests
func TestNewQuery(t *testing.T) {
	d := mock.NewDomain("testdomain")
	c := d.Class("testclass")
	s := mock.NewStore(d)

	obj1 := "object1"
	obj2 := "object2"
	q := mock.NewQuery(c, "selector", obj1, obj2)

	assert.Equal(t, c, q.Class())
	assert.Equal(t, "selector", q.Data())
	assert.Equal(t, "testdomain:testclass:selector", q.String())
	r := result.NewList()
	assert.NoError(t, s.Get(context.Background(), q, nil, r))
	assert.Equal(t, []korrel8r.Object{obj1, obj2}, r.List())
}

// Result tests
func TestResult(t *testing.T) {
	var r mock.Result

	r.Append("obj1", "obj2")
	r.Append("obj3")

	assert.Equal(t, []korrel8r.Object{"obj1", "obj2", "obj3"}, r.List())
}

// Builder tests
func TestNewBuilder(t *testing.T) {
	d1 := mock.NewDomain("domain1")

	b := mock.NewBuilder(d1, "domain2")

	assert.Equal(t, d1, b.Domain("domain1"))
	assert.Equal(t, "domain2", b.Domain("domain2").Name())

	assert.Panics(t, func() { b.Domain("nonexistent") })
}

func TestBuilder_Class(t *testing.T) {
	d := mock.NewDomain("testdomain", "class1")
	b := mock.NewBuilder(d)

	c1 := d.Class("class1")
	c2 := b.Class(c1)
	assert.Equal(t, c1, c2)

	c3 := b.Class("testdomain:class1")
	assert.Equal(t, "class1", c3.Name())
	assert.Equal(t, d, c3.Domain())

	assert.Panics(t, func() {
		b.Class(123)
	})
}

func TestBuilder_Classes(t *testing.T) {
	d := mock.NewDomain("testdomain", "class1", "class2")
	b := mock.NewBuilder(d)

	c1 := d.Class("class1")
	c2 := d.Class("class2")

	classes := b.Classes(c1, "testdomain:class2", []korrel8r.Class{c1, c2})

	expected := []korrel8r.Class{c1, c2, c1, c2}
	assert.Equal(t, expected, classes)
}

func TestBuilder_Rule(t *testing.T) {
	d := mock.NewDomain("testdomain", "start", "goal")
	b := mock.NewBuilder(d)

	rule := b.Rule("test-rule", "testdomain:start", "testdomain:goal", nil)

	assert.Equal(t, "test-rule", rule.Name())
	assert.Equal(t, "start", rule.Start()[0].Name())
	assert.Equal(t, "goal", rule.Goal()[0].Name())
}

func TestBuilder_Rules(t *testing.T) {
	d := mock.NewDomain("testdomain", "start1", "start2", "goal1", "goal2")
	b := mock.NewBuilder(d)

	args := [][]any{
		{"rule1", "testdomain:start1", "testdomain:goal1"},
		{"rule2", "testdomain:start2", "testdomain:goal2", nil},
	}

	rules := b.Rules(args)

	assert.Len(t, rules, 2)
	assert.Equal(t, "rule1", rules[0].Name())
	assert.Equal(t, "rule2", rules[1].Name())
}

func TestBuilder_Query(t *testing.T) {
	d := mock.NewDomain("testdomain", "class1")
	b := mock.NewBuilder(d)

	q := b.Query("testdomain:class1", "selector", "result1", "result2")

	assert.Equal(t, "class1", q.Class().Name())
	assert.Equal(t, "selector", q.Data())
}

// Utility function tests
func TestQuerySplit(t *testing.T) {
	tests := []struct {
		input          string
		expectedDomain string
		expectedClass  string
		expectedData   string
	}{
		{"domain:class:data", "domain", "class", "data"},
		{"domain:class", "domain", "class", ""},
		{"domain", "domain", "", ""},
		{"", "", "", ""},
		{"  domain:class:data  ", "domain", "class", "data"},
		{"domain:class:data:extra", "domain", "class", "data:extra"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input_%s", tt.input), func(t *testing.T) {
			d := mock.NewDomain(tt.expectedDomain)
			if tt.expectedClass != "" {
				query := tt.input
				if tt.expectedDomain != "" && tt.expectedClass != "" {
					parsedQuery, err := d.Query(query)
					if tt.expectedDomain == d.Name() && tt.expectedClass != "" {
						require.NoError(t, err)
						assert.Equal(t, tt.expectedClass, parsedQuery.Class().Name())
						assert.Equal(t, tt.expectedData, parsedQuery.Data())
					}
				}
			}
		})
	}
}
