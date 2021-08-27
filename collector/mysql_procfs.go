package collector

import (
	"database/sql"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	// mysqlDSN could be "root:secret@tcp(127.0.0.1:3306)".
	// Disables mysql procfs if DSN is empty.
	mysqlDSN = kingpin.Flag("collector.mysqlprocfs", "DSN for information_schema.procfs.").String()
)

// ReadProcfsFromMysql stores proc/sys files received from mysql procfs plugin on a local filesystem.
func ReadProcfsFromMysql() {
	if len(*mysqlDSN) == 0 {
		return
	}

	db, err := sql.Open("mysql", *mysqlDSN+"/information_schema?charset=utf8")
	if err != nil {
		log.Errorf("mysql_procfs collector connection problem: %s", err)
		return
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Errorf("mysql_procfs db connection close: %s", err)
		}
	}()

	rows, err := db.Query("SELECT file, contents FROM procfs")
	if err != nil {
		log.Errorf("mysql_procfs collector query error: %s", err)
		return
	}

	defer func() {
		_ = rows.Close()
		_ = rows.Err() // or modify return value
	}()

	for rows.Next() {
		var fileName string

		var contents string

		err = rows.Scan(&fileName, &contents)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Debugf("No /proc /sys files configured on mysql side.")
			} else {
				log.Warnln("mysql_procfs collector query error", err)
			}
			continue
		}
		writeProcfsFile(fileName, contents)
	}
}

// writeProcfsFile stores contents to proc/sys file at local fileName.
func writeProcfsFile(fileName string, contents string) {
	if strings.HasPrefix(fileName, "/proc") {
		fileName = procFilePath(strings.TrimPrefix(fileName, "/proc"))
	} else if strings.HasPrefix(fileName, "/sys") {
		fileName = sysFilePath(strings.TrimPrefix(fileName, "/sys"))
	} else {
		return
	}
	if err := os.MkdirAll(path.Dir(fileName), 0750); err != nil {
		log.Warnln("Couldn't create directory", err)
		return
	}
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Warnln("Couldn't create file", fileName, err)
		return
	}
	defer file.Close()
	if err = file.Truncate(0); err != nil {
		log.Warnln("Couldn't truncate file", err)
		return
	}
	if _, err = file.Seek(0, 0); err != nil {
		log.Warnln("Couldn't seek", err)
		return
	}
	if _, err = file.WriteString(contents); err != nil {
		log.Warnln("Couldn't write", err)
		return
	}
}
