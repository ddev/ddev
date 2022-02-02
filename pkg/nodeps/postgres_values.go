package nodeps

// PostgresDefaultVersion is the default Postgres version
const PostgresDefaultVersion = Postgres14

// ValidPostgresVersions is the versions of Postgres that are valid
var ValidPostgresVersions = map[string]bool{
	Postgres14: true,
	Postgres13: true,
	Postgres12: true,
	Postgres11: true,
	Postgres10: true,
}

// Postgres Versions
const (
	Postgres14 = "14"
	Postgres13 = "13"
	Postgres12 = "12"
	Postgres11 = "11"
	Postgres10 = "10"
)

// PostgresConfigFile is in-container location of postgres config
const PostgresConfigFile = "/etc/postgresql/postgresql.conf"
