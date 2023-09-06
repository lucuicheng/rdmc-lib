package lib

import "C"
import (
	"github.com/cockroachdb/pebble"
	"log"
	"strings"
)

func Get(key string) string {
	kvSourceStorePath := "/tmp/database"
	kvSourceDB, err := pebble.Open(strings.Join([]string{kvSourceStorePath}, ""), &pebble.Options{FormatMajorVersion: 8})
	if err != nil {
		log.Fatal(err)
	}
	value, closer, err := kvSourceDB.Get([]byte(key))

	if err != nil {
		log.Println(key, "fetch error", err)
		//return nil
	}
	if closer == nil {
		log.Println(key, "not found")
		//return nil
	} else {
		if err := closer.Close(); err != nil {
			log.Println(key, "close error : ", err)
			//return nil
		}
	}

	kvSourceDB.Close()

	return string(value)
}
