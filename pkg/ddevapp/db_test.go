package ddevapp

import (
	asrt "github.com/stretchr/testify/assert"
	"testing"
)

func TestDBTypeVersionFromString(t *testing.T) {
	assert := asrt.New(t)

	expectations := map[string]string{
		"9":            "postgres:9",
		"10":           "postgres:10",
		"11":           "postgres:11",
		"12":           "postgres:12",
		"13":           "postgres:13",
		"14":           "postgres:14",
		"5.5":          "mariadb:5.5",
		"5.6":          "mysql:5.6",
		"5.7":          "mysql:5.7",
		"8.0":          "mysql:8.0",
		"10.0":         "mariadb:10.0",
		"10.1":         "mariadb:10.1",
		"10.2":         "mariadb:10.2",
		"10.3":         "mariadb:10.3",
		"10.4":         "mariadb:10.4",
		"10.5":         "mariadb:10.5",
		"10.6":         "mariadb:10.6",
		"10.7":         "mariadb:10.7",
		"mariadb_10.2": "mariadb:10.2",
		"mariadb_10.3": "mariadb:10.3",
		"mysql_5.7":    "mysql:5.7",
		"mysql_8.0":    "mysql:8.0",
	}

	for input, expectation := range expectations {
		assert.Equal(expectation, dbTypeVersionFromString(input))
	}

}
