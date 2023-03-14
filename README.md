# gsqlsep

A Go implementation of Google SQL(a.k.a. ZetaSQL) statement separator.
Google SQL is the SQL dialect of Cloud Spanner, BigQuery, Apache Beam SQL ZetaSQL dialect, etc.

This is experimental and provided without warranty of any kind.


## Features

- Separate statements without syntax parsing.
  - Query syntax changes will not break user codes.
  - It is conceptually an implementation of the [Google SQL lexical structure](https://github.com/google/zetasql/blob/master/docs/lexical.md).
- Strip comments to support Cloud Spanner Admin API, which doesn't support comments in DDL.
- (Experimental) Alternative termination characters are customizable.
  - It is possible to support spanner-cli style command terminators `\G`.
    - Example: `SELECT 1\G`

## Acknowledgements

The implementation of this package is almost a fork of [spanner-cli](https://github.com/cloudspannerecosystem/spanner-cli) which is derived from [spansql](https://github.com/googleapis/google-cloud-go/tree/spanner/v1.44.0/spanner/spansql).

## References

### Backgrounds

Statement separators tend to be reinvented because there are no reusable implementations.

- https://github.com/cloudspannerecosystem/spanner-cli/pull/34

### Language Specifications

- https://github.com/google/zetasql/blob/master/docs/lexical.md
- https://cloud.google.com/bigquery/docs/reference/standard-sql/lexical
- https://cloud.google.com/spanner/docs/reference/standard-sql/lexical
- https://cloud.google.com/dataflow/docs/reference/sql/lexical
- https://beam.apache.org/documentation/dsls/sql/zetasql/lexical/