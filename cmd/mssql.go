package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/denisenkom/go-mssqldb/msdsn"
	"github.com/patrickmn/go-cache"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"rdmc/pkg"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DatabaseSQLServerCache struct {
	Database *sql.DB
	Cache    *cache.Cache
}

var GlobalDatabaseSQLServerCache = &DatabaseSQLServerCache{}

func (c *DatabaseSQLServerCache) Init(extra string, databaseInstance *sql.DB) *DatabaseSQLServerCache {
	c.Cache = cache.New(24*time.Hour, 48*time.Hour)

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

func (c *DatabaseSQLServerCache) Query(extra string, query string) ([]map[string]string, error) {

	now := time.Now()
	prefix := now.Format("2006-01-02")

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "sqlite", prefix)
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes

	cachedInstance, found := c.Cache.Get(key)
	if found {
		//log.Infof("already create one rocksdb")
		database := cachedInstance.(*sql.DB)
		//log.Infof("database sqlite [%s] mem pointer is %s\n", prefix, &database)

		rows, err := database.Query(query)

		if err != nil {
			fmt.Println(2)
			fmt.Printf("query %v\n", err)
			os.Exit(0)
			return []map[string]string{}, errors.New("query failed")
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
				return []map[string]string{}, errors.New("scan column failed")
			}
			//每一行
			row := make(map[string]string)
			for i := 0; i < length; i++ {
				row[columns[i]] = string(values[i])
			}
			list = append(list, row)
		}

		return list, nil
	}

	return []map[string]string{}, errors.New("can not get database instance")
}

func (c *DatabaseSQLServerCache) QueryCombine(extra string, filters string, query string) ([]string, error) {

	now := time.Now()
	prefix := now.Format("2006-01-02")

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "sqlite", prefix)
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes

	cachedInstance, found := c.Cache.Get(key)
	if found {
		//log.Infof("already create one rocksdb")
		database := cachedInstance.(*sql.DB)
		//log.Infof("database sqlite [%s] mem pointer is %s\n", prefix, &database)

		rows, err := database.Query(query)
		//fmt.Print(rows)

		if err != nil {
			fmt.Println(2)
			fmt.Printf("combine query %v\n", err)
			os.Exit(0)
			return []string{}, errors.New("query failed")
		}

		var list []string
		columns, _ := rows.Columns()
		columnTypes, _ := rows.ColumnTypes()

		//多少个字段
		length := len(columnTypes)
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
				return []string{}, errors.New("scan column failed")
			}
			//每一行
			//row := make(map[string]string)
			var row []string
			for i := 0; i < length; i++ {
				//fmt.Println(columnTypes[i].DatabaseTypeName(), values[i])
				if strings.Contains(columnTypes[i].DatabaseTypeName(), "CHAR") && (pkg.FilterValue(columns[i], filters) || filters == "") {
					//row[columns[i]] = string(values[i])
					row = append(row, string(columns[i])+":"+string(values[i]))
				}

			}
			list = append(list, strings.Join(row, "#"))
		}

		return list, nil
	}

	return []string{}, errors.New("can not get database instance")
}

func (c *DatabaseSQLServerCache) QueryCombineByRiver(extra string, filters string, query string) ([]string, error) {

	now := time.Now()
	prefix := now.Format("2006-01-02")

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "sqlite", prefix)
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes

	cachedInstance, found := c.Cache.Get(key)
	if found {
		//log.Infof("already create one rocksdb")
		database := cachedInstance.(*sql.DB)
		//log.Infof("database sqlite [%s] mem pointer is %s\n", prefix, &database)
		rows, err := database.Query(query)

		if err != nil {
			fmt.Println(2)
			fmt.Printf("combine query %v\n", err)
			os.Exit(0)
			return []string{}, errors.New("query failed")
		}

		var list []string
		columns, _ := rows.Columns()
		columnTypes, _ := rows.ColumnTypes()

		//多少个字段
		length := len(columnTypes)
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
				return []string{}, errors.New("scan column failed")
			}
			//每一行
			//row := make(map[string]string)
			var row []string
			for i := 0; i < length; i++ {
				//fmt.Println(columnTypes[i].DatabaseTypeName(), values[i])  增加主键显示
				if strings.Contains(columnTypes[i].DatabaseTypeName(), "CHAR") && (pkg.FilterValue(columns[i], filters) || filters == "") {
					//row[columns[i]] = string(values[i])
					row = append(row, string(columns[i])+":"+string(values[i]))
				}

			}
			list = append(list, strings.Join(row, "\t"))
		}

		return list, nil
	}

	return []string{}, errors.New("can not get database instance")
}

func CreateDatabaseSQLServer(remote bool, times int, connector pkg.Connector) *sql.DB {
	// server 启动之后，全局保持一个 db 实例

	sqlserver := pkg.Connection{
		HOST:     connector.Host,
		PORT:     connector.Port,
		USERNAME: connector.Username,
		PASSWORD: connector.Password,
		SID:      connector.Sid,
		DATABASE: connector.Database,
	}

	// 默认本地连接
	connection := ""
	//
	if remote { // server 启动之后，全局保持一个 db 实例
		// TODO 原则上无需校验
		connection = fmt.Sprintf(`server=%s\%s;user id=%s;password=%s;database=%s;trustservercertificate=true`,
			sqlserver.HOST, sqlserver.SID, sqlserver.USERNAME, sqlserver.PASSWORD, sqlserver.DATABASE)
	} else {
		connection = fmt.Sprintf(`server=%s\%s;user id=%s;password=%s;database=master;database=%s;trustservercertificate=true`,
			"localhost", sqlserver.SID, sqlserver.USERNAME, sqlserver.PASSWORD, sqlserver.DATABASE)
	}

	//log.Println(connection)

	cfg, _, _ := msdsn.Parse(connection)
	cfg.TLSConfig.MinVersion = tls.VersionTLS10

	conn := mssql.NewConnectorConfig(cfg)
	db := sql.OpenDB(conn)

	//db, err := sql.Open("sqlserver", connection) // Open the created SQLite File
	//
	//if err != nil {
	//	//fmt.Fprintln(os.Stderr, "connection create failed: ", err)
	//	fmt.Println(2)
	//	fmt.Println("connection create failed: ", err)
	//	os.Exit(0)
	//	return nil
	//}

	// 创建一个上下文对象 设置连接超时时间
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(times)*time.Second)
	defer cancel()

	// 尝试建立连接
	err := db.PingContext(ctx)
	if err != nil {
		//fmt.Fprintln(os.Stderr, "connection create failed: ", err)
		fmt.Println(2)
		fmt.Println("custom connection create failed: ", err)
		os.Exit(0)
		return nil
	}

	db.SetMaxOpenConns(350)
	db.SetMaxIdleConns(200)
	db.SetConnMaxLifetime(200)

	return db
}

func SqlServerDataRequest(host string, port int, index int, size int, schema string, table string, filters string,
	results chan<- string, wg *sync.WaitGroup) (*http.Client, *http.Request) {

	api := fmt.Sprintf("http://%s:%d/agent/atomic/detection", host, port)

	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("request [%s] error : %v\n", api, err)
			return
		}
	}()

	//------------------

	// 获取分段的数据, 按照固定长度获取 数据，假设固定长度 为默认的 500，则一批次 1-501 500-10001 ... ddr_database.ddrtable_82974
	sqlString := fmt.Sprintf(`SELECT * FROM (SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS RowID, s.* FROM %s.%s s) a order by RowID OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`,
		schema, table, index*size, size)
	result, _ := GlobalDatabaseSQLServerCache.QueryCombine("", filters, sqlString)

	// 只有 0 才需要不检测数据，直接标记完成， 上述查询迁移到 goroutine 中
	if len(result) < size && len(result) != 0 {
		log.Println(fmt.Sprintf(`part data [%d] [%s] [%d][%d]`, index, sqlString, len(result), size))
	} else if len(result) == 0 {
		results <- fmt.Sprintf("No Sending %s: %s", api, "no data")
		log.Println(fmt.Sprintf(`no data [%d] [%s] [%d][%d]`, index, sqlString, len(result), size))
		return nil, nil
	} else {
		log.Println(fmt.Sprintf(`full data [%d] [%s] [%d][%d]`, index, sqlString, len(result), size))
	}

	//------------------

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
		return nil, nil
	}

	req, err := http.NewRequest(http.MethodPost, uri.String(), strings.NewReader(strings.Join(result, "\n")))
	if err != nil {
		//fmt.Printf("run remote direct url, request create error %v\n", err)
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
		return nil, nil
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Content-Encoding", "gzip")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		//fmt.Printf("error：%v", err)
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Printf("error：%v", err)
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	}

	if string(body) == "" {
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	} else {
		//fmt.Println(pkg.ToJSONObject(string(body)))
		results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}

	return client, req
}

func SqlServerDataRequestByRiver(host string, port int, index int, size int, schema string, table string, filters string,
	results chan<- string, wg *sync.WaitGroup) (*http.Client, *http.Request) {

	api := fmt.Sprintf("http://%s:%d/predict", host, port)

	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("request [%s] error : %v\n", api, err)
			return
		}
	}()

	//------------------

	// 获取分段的数据, 按照固定长度获取 数据，假设固定长度 为默认的 500，则一批次 1-501 500-10001 ... ddr_database.ddrtable_82974
	sqlString := fmt.Sprintf(`SELECT * FROM (SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS RowID, s.* FROM %s.%s s) a order by RowID OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`,
		schema, table, index*size, size)
	result, _ := GlobalDatabaseSQLServerCache.QueryCombineByRiver("", filters, sqlString)
	//fmt.Println(result[0])
	// 只有 0 才需要不检测数据，直接标记完成， 上述查询迁移到 goroutine 中
	if len(result) < size && len(result) != 0 {
		log.Println(fmt.Sprintf(`part data [%d] [%s] [%d][%d]`, index, sqlString, len(result), size))
	} else if len(result) == 0 {
		results <- fmt.Sprintf("No Sending %s: %s", api, "no data")
		log.Println(fmt.Sprintf(`no data [%d] [%s] [%d][%d]`, index, sqlString, len(result), size))
		return nil, nil
	} else {
		log.Println(fmt.Sprintf(`full data [%d] [%s] [%d][%d]`, index, sqlString, len(result), size))
	}

	//------------------

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
		return nil, nil
	}
	bytes, _ := json.Marshal(result)

	req, err := http.NewRequest(http.MethodPost, uri.String(), strings.NewReader(string(bytes)))
	//req, err := http.NewRequest(http.MethodPost, uri.String(), strings.NewReader(strings.Join(result, "\n")))
	if err != nil {
		//fmt.Printf("run remote direct url, request create error %v\n", err)
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
		return nil, nil
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Content-Encoding", "gzip")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		//fmt.Printf("error：%v", err)
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//fmt.Printf("error：%v", err)
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	}

	if string(body) == "" {
		results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	} else {
		//fmt.Println(string(body))
		results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}

	return client, req
}

func main() {

	version := flag.Bool("v", false, "Prints current tool version")

	remote := flag.Bool("remote", false, "remote connect or not")
	engine := flag.Int("engine", 0, "detect engine type, 0=activeio, 1=ruishu")
	timeout := flag.Int("timeout", 10, "connection timeout")
	thread := flag.Int("thread", 6, "the thread")

	host := flag.String("host", "", "sqlserver host")
	port := flag.String("port", "", "sqlserver port")
	username := flag.String("username", "", "sqlserver user name")
	password := flag.String("password", "", "sqlserver user password")
	instance := flag.String("instance", "", "sqlserver instance")
	database := flag.String("database", "", "sqlserver database")

	structs := flag.Bool("structs", false, "direct query sqlserver (select)") // 结构检测
	detect := flag.Bool("detect", false, "detect sqlserver data (select)")    // 加密检测
	entropy := flag.Bool("entropy", false, "detect entropy data (select)")    // 熵值计算

	filters := flag.String("filter", pkg.SqlServerStructFilter, "the filter string")
	include := flag.String("include", "", "the include string")
	exclude := flag.String("exclude", "", "the exclude string")
	querySQL := flag.String("query", pkg.SqlServerStructSql, "the query sql string")

	schema := flag.String("schema", pkg.OracleStructSql, "the schema")
	table := flag.String("table", pkg.OracleStructSql, "the table")
	detected := flag.String("detected", "", "the detected row data")

	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - SQL Server Database Operator v%s", pkg.AppVersion))
		os.Exit(0)
	}

	start := time.Now()

	connector := &pkg.Connector{}
	connector.Host = *host         //"10.0.0.201"
	connector.Port = *port         //"1433"
	connector.Username = *username //"sa"
	connector.Password = *password //"P@ssw0rd"
	connector.Sid = *instance      //"leora"
	connector.Database = *database //"leora"

	//cstSh, _ := time.LoadLocation("Asia/Shanghai")

	GlobalDatabaseSQLServerCache.Init("", CreateDatabaseSQLServer(*remote, *timeout, *connector))

	// 实际内容的解析
	var filterList []map[string]string
	var containList []map[string]string

	if *structs {

		databases, _ := GlobalDatabaseSQLServerCache.Query("", "SELECT name as CATALOG FROM sys.databases WHERE database_id > 4 and state_desc = 'ONLINE'")

		var result []map[string]string
		for _, row := range databases {
			val, ok := row["CATALOG"]
			if ok {
				// 过滤 database
				if pkg.FilterValue(val, *filters) {
					data, _ := GlobalDatabaseSQLServerCache.Query("", "use "+val+";"+*querySQL)
					//
					for i, datum := range data {
						datum["CATALOG"] = val
						data[i] = datum
					}
					result = append(result, data...)
				}
			}
		}

		// 过滤内容
		for _, row := range result {
			val, ok := row["SCHEMANAME"]
			if ok {
				if pkg.FilterValue(val, *filters) {
					//childJson, _ := json.Marshal(job) // 序列化
					filterList = append(filterList, row)
					//fmt.Printf("%v\n", string(childJson))
				}
			}
		}

		// 包含内容
		for _, row := range filterList {
			val, ok := row["SCHEMANAME"]
			if ok {
				if pkg.ContainValue(val, *include) || *include == "" {
					//childJson, _ := json.Marshal(job) // 序列化
					containList = append(containList, row)
					//fmt.Printf("%v\n", string(childJson))
				}
			}
		}

		//fmt.Println(len(output))
		for _, o := range containList {
			fmt.Println(fmt.Sprintf("%s.%s.%s", o["CATALOG"], o["SCHEMANAME"], o["TABLENAME"]))
		}

		//fmt.Printf("Finish Tasks Cost=[%v]\n", time.Since(start))
		log.Printf("Finish Tasks Cost=[%v]\n", time.Since(start))
	}

	if *detect {

		count := 0        // 起始位置
		size := 10000 * 2 // 默认-抽取数据数

		// TODO 需要控制下最大并发数
		thread := *thread // 线程数
		var noDetected = true
		for {
			// 多个 goroutine 查询数据，每个查询数据中再实现 类似 all promise 的，获取结果
			results := make(chan string)
			var done = false
			var wg sync.WaitGroup

			for i := count * thread; i < (count+1)*thread; i++ {
				wg.Add(1)
				if *engine == 0 {
					go SqlServerDataRequest("localhost", 56790, i, size, *schema, *table, *exclude, results, &wg)
				} else {
					go SqlServerDataRequestByRiver("localhost", 56788, i, size, *schema, *table, *exclude, results, &wg)
				}
			}

			go func() {
				wg.Wait()
				close(results)
			}()

			for result := range results {
				if strings.Contains(result, "Fetched ") {
					if *engine == 0 {
						response := pkg.Response{}
						content := strings.Split(strings.TrimSpace(strings.ReplaceAll(result, "Fetched", "")), "::")
						err := json.Unmarshal([]byte(content[1]), &response) //
						if err == nil && response.Body.Detected != "" {
							// 解析 response 返回完整数据
							//contentDetails := strings.Split(content[0], ";")
							fmt.Println(1)
							fmt.Println(response.Body.Detected)
							done = true
							noDetected = false
							break
						}
					} else {
						response := pkg.RiverResponse{}
						content := strings.Split(strings.TrimSpace(strings.ReplaceAll(result, "Fetched", "")), "::")
						err := json.Unmarshal([]byte(content[1]), &response) //
						doBreak := false
						index := 0
						//fmt.Println(len(response.PredictResult))
						if err == nil && len(response.PredictResult) != 0 {
							for i, num := range response.PredictResult {
								if num == 1 {
									index = i
									doBreak = true
									break
								}
							}
						}
						if doBreak {
							//log.Println(response.PredictResult)

							// 解析 response 返回完整数据
							//contentDetails := strings.Split(content[0], ";")
							fmt.Println(1)
							position := strings.Split(content[0], ";")
							start, _ := strconv.Atoi(position[1])
							// TODO  再次查询数据库，获取真是信息
							fmt.Println(start + index)
							done = true
							break
						}
					}
				} else if strings.Contains(result, "No Sending") {
					done = true
					continue
				}
			}
			//等待完成

			count++
			time.Sleep(100 * time.Millisecond)

			//fmt.Println("run count :", count)
			if done { // 是否存在 分页查询不满，不满认为已经可以结束了
				break
			}

		}

		if noDetected {
			fmt.Println(0)
		}

		log.Printf("Finish Tasks Cost=[%v]\n", time.Since(start))
	}

	if *entropy && *engine == 0 {
		//  基础信息获取 - 主键信息
		primaryKeySqlString := fmt.Sprintf(`SELECT Col.Column_Name as PK
from INFORMATION_SCHEMA.TABLE_CONSTRAINTS Tab,
     INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE Col
WHERE Col.Constraint_Name = Tab.Constraint_Name
  	AND Col.Table_Name = Tab.Table_Name
  	AND Tab.Constraint_Type = 'PRIMARY KEY'
  	AND Col.table_schema = '%s'
  	AND Col.Table_Name = '%s'`, *schema, *table)
		//fmt.Println(primaryKeySqlString)

		result, _ := GlobalDatabaseSQLServerCache.Query("", primaryKeySqlString)
		primaryKeyAliasSql := ""
		if len(result) > 0 {
			primaryKeyAliasSql = fmt.Sprintf("%s as ADMCID, ", result[0]["PK"])
		}
		//fmt.Println(primaryKeyAliasSql)

		// 计算熵值，IDS 统计计算
		detected := *detected
		detectedColumnValues := strings.Split(detected, "#")
		var columns []string
		var columnsCombine []string
		var whereCondition []string
		for _, columnNameValue := range detectedColumnValues {
			columnNameValueArray := strings.SplitN(columnNameValue, ":", 2)
			columnName := columnNameValueArray[0]
			columnValue := columnNameValueArray[1]
			// 列名需要反引号 转义特殊处理
			columns = append(columns, fmt.Sprintf("%s", columnName))
			columnsCombine = append(columnsCombine, fmt.Sprintf("ISNULL(%s + ', ', '')", columnName))

			if columnValue == "" {
				whereCondition = append(whereCondition, fmt.Sprintf("( %s = '' or %s is null )", columnName, columnName))
			} else {
				columnValue = strings.Replace(columnValue, "'", "''", -1)
				whereCondition = append(whereCondition, fmt.Sprintf("%s = '%s'", columnName, columnValue))
			}
		}
		//fmt.Println(whereCondition)

		sql := fmt.Sprintf(`SELECT a.ROWID as ADMC_ROW_NUMBER
FROM (SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS ROWID,
             %s
      FROM %s.%s S
     ) a
WHERE %s`, strings.Join(columns, ", "), *schema, *table, strings.Join(whereCondition, " and "))
		//fmt.Println(sql)

		result, _ = GlobalDatabaseSQLServerCache.Query("", sql)

		if len(result) <= 0 {
			fmt.Println(2)
			fmt.Println("Data detection does not exist")
			return
		}

		breakPoint, _ := strconv.Atoi(result[0]["ADMC_ROW_NUMBER"]) // TODO 判断位置选项
		size := 500
		offset := breakPoint - 250

		if offset < 0 {
			offset = 0
			breakPoint = 251
		}

		sql = fmt.Sprintf(`SELECT %s %s as COMBINE FROM (SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS RowID, s.* FROM %s.%s s) a order by RowID OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`,
			primaryKeyAliasSql, strings.Join(columnsCombine, "+"), *schema, *table, offset, size)
		//fmt.Println(sql)

		// 循环 500 计算 实际熵值与对应 id
		var entropy []string
		var ids []string
		result, _ = GlobalDatabaseSQLServerCache.Query("", sql)
		for _, row := range result {
			entropy = append(entropy, strconv.Itoa(pkg.Shannon(row["COMBINE"])))
			val, ok := row["ADMCID"]
			if ok {
				ids = append(ids, val)
			}
		}

		fmt.Println(1)                          // 检测状态
		fmt.Println(strings.Join(entropy, ",")) // 熵值
		fmt.Println(strings.Join(ids, ","))     // ids

		// 返回位置信息
		fmt.Println(500)              //rowCount
		fmt.Println(breakPoint - 250) //rowStart
		fmt.Println(breakPoint + 249) //rowEnd
		fmt.Println(detected)
	}

	if *entropy && *engine == 1 {
		//  基础信息获取 - 主键信息
		primaryKeySqlString := fmt.Sprintf(`SELECT Col.Column_Name as PK
from INFORMATION_SCHEMA.TABLE_CONSTRAINTS Tab,
     INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE Col
WHERE Col.Constraint_Name = Tab.Constraint_Name
  	AND Col.Table_Name = Tab.Table_Name
  	AND Tab.Constraint_Type = 'PRIMARY KEY'
  	AND Col.table_schema = '%s'
  	AND Col.Table_Name = '%s'`, *schema, *table)
		//fmt.Println(sql)

		result, _ := GlobalDatabaseSQLServerCache.Query("", primaryKeySqlString)
		primaryKeyAliasSql := ""
		if len(result) > 0 {
			primaryKeyAliasSql = fmt.Sprintf("%s as ADMCID, ", result[0]["PK"])
		}

		//  基础信息获取 - 列名信息
		columnSqlString := fmt.Sprintf(`SELECT a.column_name as COLUMN_NAME
FROM INFORMATION_SCHEMA.COLUMNS a
where a.data_type like '%%char%%'
  	and a.table_name = '%s'
 	and a.table_schema = '%s'`, *table, *schema)
		//fmt.Println(columnSqlString)

		result, _ = GlobalDatabaseSQLServerCache.Query("", columnSqlString)
		var columns []string
		var columnsCombine []string
		if len(result) > 0 {
			for i := range result {
				columns = append(columns, result[i]["COLUMN_NAME"])
				columnsCombine = append(columnsCombine, fmt.Sprintf("ISNULL(%s + ', ', '')", result[i]["COLUMN_NAME"]))
			}
		}

		// 计算熵值，IDS 统计计算
		breakPoint, _ := strconv.Atoi(*detected)
		var one_result string

		sql := fmt.Sprintf(`SELECT %s FROM (SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS RowID, s.* FROM %s.%s s) a order by RowID OFFSET %d ROWS FETCH NEXT 1 ROWS ONLY`, strings.Join(columns, ","), *schema, *table, breakPoint-1)
		//fmt.Println(sql)

		result, _ = GlobalDatabaseSQLServerCache.Query("", sql)

		if len(result) <= 0 {
			fmt.Println(2)
			fmt.Println("Data detection does not exist")
			return
		} else {
			var row []string
			for k, v := range result[0] {
				row = append(row, string(k)+":"+string(v))
			}
			one_result = strings.Join(row, "#")
		}

		//breakPoint, _ := strconv.Atoi(result[0]["ADMC_ROW_NUMBER"]) // TODO 判断位置选项
		size := 500
		offset := breakPoint - 250

		if offset < 0 {
			offset = 0
			breakPoint = 251
		}

		sql = fmt.Sprintf(`SELECT %s %s as COMBINE FROM (SELECT ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS RowID, s.* FROM %s.%s s) a order by RowID OFFSET %d ROWS FETCH NEXT %d ROWS ONLY`,
			primaryKeyAliasSql, strings.Join(columnsCombine, ","), *schema, *table, offset, size)
		//fmt.Println(sql)

		// 循环 500 计算 实际熵值与对应 id
		var entropy []string
		var ids []string
		result, _ = GlobalDatabaseSQLServerCache.Query("", sql)
		for _, row := range result {
			entropy = append(entropy, strconv.Itoa(pkg.Shannon(row["COMBINE"])))
			val, ok := row["ADMCID"]
			if ok {
				ids = append(ids, val)
			}
		}

		fmt.Println(1)                          // 检测状态
		fmt.Println(strings.Join(entropy, ",")) // 熵值
		fmt.Println(strings.Join(ids, ","))     // ids

		// 返回位置信息
		fmt.Println(500)              //rowCount
		fmt.Println(breakPoint - 250) //rowStart
		fmt.Println(breakPoint + 249) //rowEnd
		//fmt.Println(detected)
		fmt.Println(one_result)
	}

	GlobalDatabaseSQLServerCache.Database.Close()
}
