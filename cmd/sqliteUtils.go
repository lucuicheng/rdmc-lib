package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/patrickmn/go-cache"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func toJSONObject(result string) map[string]interface{} {
	var data map[string]interface{} // 对象
	//var dats []map[string]interface{} // 数组

	err := json.Unmarshal([]byte(result), &data)
	if err != nil {
		fmt.Println(err.Error())
	}
	return data
}

func insertData(host string, port int, paramsBytes []byte) (*http.Client, *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("request error : %v\n", err)
			return
		}
	}()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}
	api := fmt.Sprintf("http://%s:%d/agent/atomic/job", host, port)

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {
		fmt.Println(err.Error())
		return nil, nil
	}

	req, err := http.NewRequest(http.MethodPost, uri.String(), strings.NewReader(string(paramsBytes)))
	if err != nil {
		//fmt.Printf("run remote direct url, request create error %v\n", err)
		return nil, nil
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		//fmt.Printf("error：%v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Printf("error：%v", err)
	}

	if string(body) == "" {
		//fmt.Println(errors.New("no response"))
	} else {
		fmt.Println(toJSONObject(string(body)))
	}

	return client, req
}

func updateData(host string, port int, paramsBytes []byte) (*http.Client, *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("request error : %v\n", err)
			return
		}
	}()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}
	api := fmt.Sprintf("http://%s:%d/agent/atomic/job_status", host, port)

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {
		fmt.Println(err.Error())
		return nil, nil
	}

	req, err := http.NewRequest(http.MethodPut, uri.String(), strings.NewReader(string(paramsBytes)))
	if err != nil {
		//fmt.Printf("run remote direct url, request create error %v\n", err)
		return nil, nil
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		//fmt.Printf("error：%v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Printf("error：%v", err)
	}

	if string(body) == "" {
		//fmt.Println(errors.New("no response"))
	} else {
		//fmt.Println(pkg.ToJSONObject(string(body)))
	}

	return client, req
}

type DatabaseSQLiteCache struct {
	Database *sql.DB
	Cache    *cache.Cache
}

func CreateDatabaseSQLite() *sql.DB {
	// server 启动之后，全局保持一个 db 实例
	databasePath := "./sqlite-database.db"
	sqliteDatabase, _ := sql.Open("sqlite3", databasePath) // Open the created SQLite File

	sqliteDatabase.SetMaxOpenConns(350)
	sqliteDatabase.SetMaxIdleConns(200)
	sqliteDatabase.SetConnMaxLifetime(200)

	return sqliteDatabase
}

func (c *DatabaseSQLiteCache) Init(extra string, databaseInstance *sql.DB) *DatabaseSQLiteCache {
	c.Cache = cache.New(30*time.Minute, 60*time.Minute)

	// 使用系统日志
	//log := logger.SystemAppender().
	//	WithField("component", "SYSTEM").
	//	WithField("category", "DATABASE")

	now := time.Now()
	prefix := now.Format("2006-01-02")

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "sqlite", prefix)
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes

	// Set the value of the key "foo" to "bar", with the default expiration time
	cachedSQLite, found := c.Cache.Get(key)
	if found {
		//log.Infof("already create one rocksdb")
		database := cachedSQLite.(*sql.DB)
		//log.Infof("database sqlite [%s] mem pointer is %s\n", prefix, &database)
		c.Database = database
		//return db, nil
	}

	//log.Infof("cached logger not found, start to create new logger[%s] and set it to cache", key)

	c.Cache.Set(key, databaseInstance, cache.DefaultExpiration)
	c.Database = databaseInstance

	return c
}

func (c *DatabaseSQLiteCache) GetObject() *DatabaseSQLiteCache {

	now := time.Now()
	prefix := now.Format("2006-01-02")

	extra := ""

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "sqlite", prefix)

	//log := logger.SystemAppender().
	//	WithField("component", "SYSTEM").
	//	WithField("category", "DATABASE")

	cachedSQLite, found := c.Cache.Get(key)
	if found {
		//log.Infof("already create one rocksdb")
		//fmt.Println("use cached sqlite db")
		database := cachedSQLite.(*sql.DB)
		//log.Infof("found，database sqlite [%s] mem pointer is %v", prefix, &database)
		c.Database = database
		//return db, nil
	} else {
		// 重新初始化
		fmt.Println("recreate sqlite db")
		c.Cache.Flush() // 先清空，再重新初始化，确保只有一个有效缓存
		return c.Init("", CreateDatabaseSQLite())
	}

	return c
}

var GlobalDatabaseSQLiteCache = &DatabaseSQLiteCache{}

type Job struct {
	id         int
	source     string
	key        string
	link       string
	status     int
	name       string
	createDate string
}

func main() {

	insert := flag.Bool("insert", false, "request post type (insert)")
	update := flag.Bool("update", false, "request put type (update)")
	query := flag.Bool("query", false, "direct query database (select)")

	source := flag.String("source", "", "the source, host address, eg:10.0.0.103")
	key := flag.String("key", "", "the job name, eg:JOB_XXXX")
	application := flag.String("application", "", "the application id, eg:9170564")
	name := flag.String("name", "", "the application name, eg:LEORA")

	sqlString := flag.String("sql", "select * from job", "the query sqlString, eg:sqlString")

	id := flag.Int("id", 1, "the id")
	status := flag.Int("status", 100, "the status")
	date := flag.Int64("date", 100, "the status")

	flag.Parse()

	start := time.Now()

	params := make(map[string]map[string]interface{})
	_data := make(map[string]interface{})

	//cstSh, _ := time.LoadLocation("Asia/Shanghai")

	if *insert {
		fmt.Printf("insert=[%v] data=[%s]\n", *insert, *name)

		_data["source"] = *source
		_data["key"] = *key
		_data["application"] = *application
		_data["status"] = 100
		_data["name"] = *name
		params["data"] = _data

		paramsBytes, _ := json.Marshal(params)
		insertData("localhost", 56790, paramsBytes)
	}

	if *update {
		fmt.Printf("update=[%v] id=[%d] status=[%d]\n", *update, *id, *status)

		_data["id"] = *id
		_data["status"] = *status
		_data["updateDate"] = date
		params["data"] = _data

		paramsBytes, _ := json.Marshal(params)
		updateData("localhost", 56790, paramsBytes)
	}

	if *query {
		cached := GlobalDatabaseSQLiteCache.Init("", CreateDatabaseSQLite())
		//cached := GlobalDatabaseSQLiteCache.GetObject()
		sqliteDatabase := cached.Database

		rows, err := sqliteDatabase.Query(*sqlString)

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

	fmt.Printf("Finish Tasks Cost=[%v]\n", time.Since(start))
}
