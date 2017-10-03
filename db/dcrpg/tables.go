package dcrpg

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	"github.com/dcrdata/dcrdata/db/dcrpg/internal"
)

var createTableStatements = map[string]string{
	"blocks":       internal.CreateBlockTable,
	"transactions": internal.CreateTransactionTable,
	"vins":         internal.CreateVinTable,
	"vouts":        internal.CreateVoutTable,
	"block_chain":  internal.CreateBlockPrevNextTable,
}

var createTypeStatements = map[string]string{
	"vin_t":  internal.CreateVinType,
	"vout_t": internal.CreateVoutType,
}

func TableExists(db *sql.DB, tableName string) (bool, error) {
	rows, err := db.Query(`select relname from pg_class where relname = $1`,
		tableName)
	if err == nil {
		defer rows.Close()
		return rows.Next(), nil
	}
	return false, err
}

func DropTables(db *sql.DB) {
	for tableName := range createTableStatements {
		fmt.Printf("DROPPING the \"%s\" table.\n", tableName)
		if err := dropTable(db, tableName); err != nil {
			fmt.Println(err)
		}
	}

	_, err := db.Exec(`DROP TYPE IF EXISTS vin;`)
	if err != nil {
		fmt.Println(err)
	}
}

func dropTable(db *sql.DB, tableName string) error {
	_, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s;`, tableName))
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func CreateTypes(db *sql.DB) error {
	var err error
	for typeName, createCommand := range createTypeStatements {
		var exists bool
		exists, err = TypeExists(db, typeName)
		if err != nil {
			return err
		}
		fmt.Printf("Does the \"%s\" type exist? %v\n", typeName, exists)
		if !exists {
			fmt.Printf("Creating the \"%s\" type.\n", typeName)
			_, err = db.Exec(createCommand)
			if err != nil {
				return err
			}
			_, err = db.Exec(fmt.Sprintf(`COMMENT ON TYPE %s
				IS 'v1';`, typeName))
			if err != nil {
				return err
			}
		}
	}
	return err
}

func TypeExists(db *sql.DB, tableName string) (bool, error) {
	rows, err := db.Query(`select typname from pg_type where typname = $1`,
		tableName)
	if err == nil {
		defer rows.Close()
		return rows.Next(), nil
	}
	return false, err
}

func CreateTables(db *sql.DB) error {
	var err error
	for tableName, createCommand := range createTableStatements {
		var exists bool
		exists, err = TableExists(db, tableName)
		if err != nil {
			return err
		}
		fmt.Printf("Does the \"%s\" table exist? %v\n", tableName, exists)
		if !exists {
			fmt.Printf("Creating the \"%s\" table.\n", tableName)
			_, err = db.Exec(createCommand)
			if err != nil {
				return err
			}
			_, err = db.Exec(fmt.Sprintf(`COMMENT ON TABLE %s
				IS 'v1';`, tableName))
			if err != nil {
				return err
			}
		}
	}
	return err
}

func TableVersions(db *sql.DB) map[string]int32 {
	versions := map[string]int32{}
	for tableName := range createTableStatements {
		Result := db.QueryRow(`select obj_description($1::regclass);`, tableName)
		var s string
		v := int(-1)
		if Result != nil {
			Result.Scan(&s)
			re := regexp.MustCompile(`^v(\d+)$`)
			subs := re.FindStringSubmatch(s)
			if len(subs) > 1 {
				var err error
				v, err = strconv.Atoi(subs[1])
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		versions[tableName] = int32(v)
	}
	return versions
}
