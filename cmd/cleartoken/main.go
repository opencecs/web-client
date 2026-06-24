package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "users.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	res, err := db.Exec("DELETE FROM settings WHERE key = ?", "device_auth_pass")
	if err != nil {
		log.Fatal(err)
	}
	affected, _ := res.RowsAffected()
	fmt.Printf("已清除 device_auth_pass，影响行数: %d\n", affected)

	rows, err := db.Query("SELECT key, value FROM settings ORDER BY key")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fmt.Println("\n当前 settings 表内容:")
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		if len(v) > 40 { v = v[:40] + "..." }
		fmt.Printf("  %s = %s\n", k, v)
	}
}
