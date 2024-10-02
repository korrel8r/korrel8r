// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package domain provides standard tests and benchmarks to run against every domain for "korrel8r-fitness".
//
// To use the tests in a new domain package:
//   - Create sub-directory "testdata/domain_test.yaml" containing
//     -
//
// and provide a query that returns BatchLen objects.
// It should have test functions TestDomain and BenchmarkDomain that call [domain.Test] and [domain.Benchmark].
package domain
