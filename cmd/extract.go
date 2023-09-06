package main

import (
	"flag"
	"fmt"
	"github.com/cockroachdb/pebble"
	"log"
	"os"
	"rdmc/internal/extract"
	"rdmc/pkg"
	"strings"
	"sync/atomic"
	"time"
)

var atomicCountAll int64

func init() {

}

func main() {

	origin := flag.Bool("origin", false, "")
	fetch := flag.Bool("fetch", false, "")
	download := flag.Bool("download", false, "")
	duplicate := flag.Bool("duplicate", false, "")
	permissions := flag.Bool("permissions", false, "")
	suffix := flag.Bool("suffix", false, "")
	content := flag.Bool("content", false, "")

	stat := flag.Bool("stat", false, "")

	root := flag.String("root", "/", "The md5 filteredCount file output path")
	count := flag.Int64("count", 0, "the extract/scan data, 0 is unlimited")
	task := flag.String("task", "Task_0000", "The md5 filteredCount file output path")
	ids := flag.String("id", "", "file need sync id(s)")
	folderId := flag.Int("folder", 1, "root folder id")

	version := flag.Bool("v", false, "Prints current tool version")

	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - FRS Scan v%s", pkg.AppVersion))
		os.Exit(0)
		return
	}

	start := time.Now()

	extract.GlobalDatabaseKeyValueCache.Init("", extract.GlobalDatabaseKeyValueCache.CreateDatabaseKeyValue(*task)) // 每一个新的任务，需要一个新的库，除非指定同样的 cache 覆盖

	if *origin {

		// echo 3 > /proc/sys/vm/drop_caches 清空缓存再执行 真实的查询
		/**
		{"access":"-rw-r--r-- 0644 644","ctime":"2023-05-29 10:09:44","gid":0,"mtime":"2023-05-29 10:09:44","nio":8243523,"size":3638,"uid":0}
		*/
		log.Println(*root)
		countNum := extract.Origin(*root, *task, *count)
		fmt.Printf("Count:%d\n", countNum)
		log.Printf("Finish Extract Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(start), countNum)

		kvSourceDB := extract.GlobalDatabaseKeyValueCache.Database
		if err := kvSourceDB.Close(); err != nil {
			log.Println("target close err : ", err)
		}

		return
	}

	if *fetch {

		log.Println(*root)
		countNum := extract.Fetch(*root, *folderId, strings.Split(*ids, ","), *task, *count)
		fmt.Printf("Count:%d\n", countNum)
		log.Printf("Finish Fetch Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(start), countNum)

		kvSourceDB := extract.GlobalDatabaseKeyValueCache.Database
		if err := kvSourceDB.Close(); err != nil {
			log.Println("target close err : ", err)
		}

		return
	}

	if *download {
		log.Println(*root)
		countNum := extract.Download(*root, *folderId, strings.Split(*ids, ","), *task, *count)
		fmt.Printf("Count:%d\n", countNum)
		log.Printf("Finish Download Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(start), countNum)

		kvSourceDB := extract.GlobalDatabaseKeyValueCache.Database
		if err := kvSourceDB.Close(); err != nil {
			log.Println("target close err : ", err)
		}
	}

	if *duplicate {
		log.Println(*root)
		countNum := extract.Duplicated(*task, *count) //547432
		log.Printf("Finish Duplicate Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(start), countNum)

		kvSourceDB := extract.GlobalDatabaseKeyValueCache.Database
		if err := kvSourceDB.Close(); err != nil {
			log.Println("target close err : ", err)
		}
	}

	if *permissions {
		log.Println(*root)
		countNum := extract.Permissions(*task, *count) //547432
		log.Printf("Finish Permissions Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(start), countNum)

		kvSourceDB := extract.GlobalDatabaseKeyValueCache.Database
		if err := kvSourceDB.Close(); err != nil {
			log.Println("target close err : ", err)
		}
	}

	if *suffix {
		log.Println(*root)
		countNum := extract.Suffix(*task, *count) //547432
		log.Printf("Finish Suffix Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(start), countNum)

		kvSourceDB := extract.GlobalDatabaseKeyValueCache.Database
		if err := kvSourceDB.Close(); err != nil {
			log.Println("target close err : ", err)
		}
	}

	if *content {
		log.Println(*root)
		countNum := extract.Content(*task, *count) //547432
		log.Printf("Finish Content Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(start), countNum)

		kvSourceDB := extract.GlobalDatabaseKeyValueCache.Database
		if err := kvSourceDB.Close(); err != nil {
			log.Println("target close err : ", err)
		}
	}

	if *stat {
		//metadata := extract.StatMetadata("/media/psf/Share/lib/scan")
		//log.Println(metadata)
		//log.Printf("Finish Extract Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(start), 0)

		kvSourceDB := extract.GlobalDatabaseKeyValueCache.Database
		kvBatch := kvSourceDB.NewIndexedBatch()
		// 遍历数据库
		iter := kvBatch.NewIter(nil) // prefixIterOptions([]byte(""))
		for iter.First(); iter.Valid(); iter.Next() {
			// Only keys beginning with "prefix" will be visited.
			keyBytes := iter.Key()
			valueBytes, err := iter.ValueAndErr()

			atomic.AddInt64(&atomicCountAll, 1)
			if err != nil {
				//log.Errorf("fetch value bytes %v", err)
				continue
			}
			//var metadata extract.Metadata
			//json.Unmarshal(valueBytes, &metadata)
			fmt.Println(string(keyBytes), string(valueBytes))
		}

		iter.Close()
		kvBatch.Commit(pebble.Sync)
		kvBatch.Close()
	}
}
