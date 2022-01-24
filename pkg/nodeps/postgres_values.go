package nodeps

// PostgresDefaultVersion is the default Postgres version
const PostgresDefaultVersion = Postgres14

// ValidPostgresVersions is the versions of Postgres that are valid
var ValidPostgresVersions = map[string]bool{
	Postgres14: true,
	Postgres13: true,
	Postgres12: true,
}

// Postgres Versions
const (
	Postgres14 = "14"
	Postgres13 = "13"
	Postgres12 = "12"
)
