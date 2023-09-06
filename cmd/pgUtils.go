package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/patrickmn/go-cache"
	"time"
)

type DatabasePostgresQLCache struct {
	Database *sql.DB
	Cache    *cache.Cache
}

type DatabaseConnection struct {
	HOST      string
	PORT      string
	DATABASE  string
	USERNAME  string
	PASSWORD  string
	CHARSET   string
	PARSETIME string
	SID       string
}

func CreateDatabasePG() *sql.DB {
	pg := DatabaseConnection{
		HOST:     "127.0.0.1",
		PORT:     "25432",
		USERNAME: "root",
		PASSWORD: "123456",
		DATABASE: "adms",
	}
	connStr := "postgresql://" + pg.USERNAME + ":" + pg.PASSWORD + "@" + pg.HOST + ":" + pg.PORT + "/" + pg.DATABASE + "?sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		fmt.Printf("connect DB failed, err:%v\n", err)
		return nil
	}

	db.SetMaxOpenConns(350)
	db.SetMaxIdleConns(200)
	db.SetConnMaxLifetime(200)

	return db
}

func (c *DatabasePostgresQLCache) Init(extra string, databaseInstance *sql.DB) *DatabasePostgresQLCache {
	c.Cache = cache.New(30*time.Minute, 60*time.Minute)

	// 使用系统日志
	//log := logger.SystemAppender().
	//	WithField("component", "SYSTEM").
	//	WithField("category", "DATABASE")

	now := time.Now()
	prefix := now.Format("2006-01-02")

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "postgresql", prefix)
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes

	// Set the value of the key "foo" to "bar", with the default expiration time
	cached, found := c.Cache.Get(key)
	if found {
		//log.Infof("already create one rocksdb")
		database := cached.(*sql.DB)
		//log.Infof("database postgresql [%s] mem pointer is %s\n", prefix, &database)
		c.Database = database
		//return db, nil
	}

	//log.Infof("cached logger not found, start to create new logger[%s] and set it to cache", key)

	c.Cache.Set(key, databaseInstance, cache.DefaultExpiration)
	c.Database = databaseInstance

	return c
}

func (c *DatabasePostgresQLCache) GetObject() *DatabasePostgresQLCache {

	now := time.Now()
	prefix := now.Format("2006-01-02")

	extra := ""

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "postgresql", prefix)

	//log := logger.SystemAppender().
	//	WithField("component", "SYSTEM").
	//	WithField("category", "DATABASE")

	cached, found := c.Cache.Get(key)
	if found {
		//log.Infof("already create one rocksdb")
		//fmt.Println("use cached postgresql db")
		database := cached.(*sql.DB)
		//log.Info("found，database postgresql [%s] mem pointer is %v", prefix, &database)
		c.Database = database
		//return db, nil
	} else {
		// 重新初始化
		fmt.Println("recreate postgresql db")
		c.Cache.Flush() // 先清空，再重新初始化，确保只有一个有效缓存
		return c.Init("", CreateDatabasePG())
	}

	return c
}

var GlobalDatabasePostgresQLCache = &DatabasePostgresQLCache{}

//  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pg_tool test_pg.go
func main() {

	add := flag.Bool("add", false, "add some manual data at some table (insert)")
	update := flag.Bool("update", false, "update some manual data at some table (update)")
	remove := flag.Bool("remove", false, "remove some manual data at some table (delete)")
	query := flag.Bool("query", false, "direct query database (select)")
	clear := flag.Bool("clear", false, "direct clear database (truncate)")

	sqlString := flag.String("sql", "", "the query sqlString, eg: select * from file_check_task ")
	schema := flag.String("schema", "public", "the schema you want to operate")
	table := flag.String("table", "", "the table you want to operate")

	flag.Parse()

	start := time.Now()

	cached := GlobalDatabasePostgresQLCache.Init("", CreateDatabasePG())
	pgInstance := cached.Database

	if *add {

	}

	if *update {

	}

	if *remove {

	}

	if *query {
		//sqlString := "select * from adms_detection_record"
		if *sqlString == "" {
			fmt.Println("no query sql string, can't do operate query")
			return
		}
		rows, err := pgInstance.Query(*sqlString)

		if err != nil {
			fmt.Printf("query %v\n", err)
			return
		}

		var list []map[string]string
		columns, _ := rows.Columns()

		//多少个字段
		length := len(columns)
		//每一行字段的值
		values := make([]sql.RawBytes, length) //内存位置
		//保存的是values的内存地址
		pointer := make([]interface{}, length) //指定长度的数组
		//
		for i := 0; i < length; i++ {
			pointer[i] = &values[i]
		}
		defer rows.Close()

		for rows.Next() {
			//把参数展开，把每一行的值存到指定的内存地址去，循环覆盖，values也就跟着被赋值了
			err := rows.Scan(pointer...)
			if err != nil {
				fmt.Println(err)
				return
			}
			//每一行
			row := make(map[string]string)
			for i := 0; i < length; i++ {
				row[columns[i]] = string(values[i])
			}
			list = append(list, row)
		}

		//for row.Next() {
		//	job := Job{} // Iterate and fetch the records from result cursor
		//	row.Scan(&job.id, &job.name)
		//	jobs = append(jobs, job)
		//}

		for _, job := range list {
			childJson, _ := json.Marshal(job) // 序列化
			fmt.Printf("%v\n", string(childJson))
		}
	}

	if *clear {
		if *schema == "" || *table == "" {
			fmt.Println("no schema or table, can't do operate truncate")
			return
		}

		// TODO 清空之前需要保存原始数据

		// 执行 TRUNCATE 命令清空表
		_, err := pgInstance.Exec(fmt.Sprintf(`TRUNCATE TABLE %s.%s`,*schema, *table))
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Table truncated successfully")
	}

	fmt.Printf("Finish Task Cost=[%v]\n", time.Since(start))
}
