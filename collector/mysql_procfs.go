package collector

import (
	"database/sql"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"path"
	"strings"
)

var (
	// e.g. root:secret@tcp(127.0.0.1:13001)
	mysqlDSN = flag.String("collector.mysqlprocfs", "", "DSN for information_schema.procfs.")
)

func ReadProcfsFromMysql() {
	if len(*mysqlDSN) == 0 {
		return
	}

	db, err := sql.Open("mysql", *mysqlDSN+"/information_schema?charset=utf8")
	checkErr(err)
	rows, err := db.Query("SELECT file, contents FROM procfs")
	checkErr(err)
	for rows.Next() {
		var fileName string
		var contents string
		err = rows.Scan(&fileName, &contents)
		checkErr(err)
		if strings.HasPrefix(fileName, "/proc") {
			fileName = procFilePath(strings.TrimPrefix(fileName, "/proc"))
		} else if strings.HasPrefix(fileName, "/sys") {
			fileName = sysFilePath(strings.TrimPrefix(fileName, "/sys"))
		} else {
			continue
		}
		os.MkdirAll(path.Dir(fileName), 0777)
		file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
		file.Truncate(0)
		file.Seek(0, 0)

		if err != nil {
			return
		}
		defer file.Close()

		file.WriteString(contents)
	}
	db.Close()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
