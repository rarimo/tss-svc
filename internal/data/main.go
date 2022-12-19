package data

//go:generate xo schema "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" -o ./ --single=schema.xo.go --src templates
//go:generate xo schema "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" -o pg --single=schema.xo.go --src=pg/templates --go-context=both
