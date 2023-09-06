package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/godror/godror"
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

type DatabaseOracleCache struct {
	Database *sql.DB
	Cache    *cache.Cache
}

var GlobalDatabaseOracleCache = &DatabaseOracleCache{}

func (c *DatabaseOracleCache) Init(extra string, databaseInstance *sql.DB) *DatabaseOracleCache {
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
	cachedInstance, found := c.Cache.Get(key)
	if found {
		//log.Infof("already create one rocksdb")
		database := cachedInstance.(*sql.DB)
		//log.Infof("database sqlite [%s] mem pointer is %s\n", prefix, &database)
		c.Database = database
		//return db, nil
	}

	//log.Infof("cached logger not found, start to create new logger[%s] and set it to cache", key)

	c.Cache.Set(key, databaseInstance, cache.DefaultExpiration)
	c.Database = databaseInstance

	return c
}

func (c *DatabaseOracleCache) Query(extra string, query string) ([]map[string]string, error) {

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

func (c *DatabaseOracleCache) QueryCombine(extra string, filters string, query string) ([]string, error) {

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

		rows, err := database.Query(query, godror.ArraySize(5000), godror.FetchArraySize(10000))

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
			list = append(list, strings.Join(row, "#"))
		}

		return list, nil
	}

	return []string{}, errors.New("can not get database instance")
}

func (c *DatabaseOracleCache) QueryCombineByRiver(extra string, filters string, query string) ([]string, error) {

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

		rows, err := database.Query(query, godror.ArraySize(5000), godror.FetchArraySize(10000))

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
					row = append(row, string(values[i]))
				}

			}
			list = append(list, strings.Join(row, "\t"))
		}

		return list, nil
	}

	return []string{}, errors.New("can not get database instance")
}

func CreateDatabaseOracle(remote bool, times int, connector pkg.Connector) *sql.DB {
	// server 启动之后，全局保持一个 db 实例

	oracle := pkg.Connection{
		HOST:     connector.Host,
		PORT:     connector.Port,
		USERNAME: connector.Username,
		PASSWORD: connector.Password,
		SID:      connector.Sid,
	}

	// 默认本地连接
	os.Setenv("LANG", "en_US.utf-8")
	connection := "/ as sysdba"
	//
	if remote { // connector != (Connector{})
		// TODO 原则上无需校验 ?connect_timeout=%d
		//connection = fmt.Sprintf("user=\"%s\" password=\"%s\" connectString=\"%s:%s/%s\" sysdba=true timezone=\"+00:00\"",
		//	oracle.USERNAME, oracle.PASSWORD, oracle.HOST, oracle.PORT, oracle.SID)

		connection = fmt.Sprintf("user=\"%s\" password=\"%s\" connectString=\"%s:%s/%s\" timezone=\"+00:00\"",
			oracle.USERNAME, oracle.PASSWORD, oracle.HOST, oracle.PORT, oracle.SID)

	} else {
		connection = fmt.Sprintf("%s/%s", oracle.USERNAME, oracle.PASSWORD)

		os.Setenv("ORACLE_SID", oracle.SID)
	}

	log.Println(connection)

	db, err := sql.Open("godror", connection) // Open the created SQLite File

	if err != nil {
		//fmt.Fprintln(os.Stderr, "connection create failed: ", err)
		fmt.Println(2)
		fmt.Println("connection create failed: ", err)
		os.Exit(0)
		return nil
	}

	// 创建一个上下文对象
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(times)*time.Second)
	defer cancel()

	// 尝试建立连接
	err = db.PingContext(ctx)
	if err != nil {
		//fmt.Fprintln(os.Stderr, "connection failed: ", err)
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

func OracleDataRequest(host string, port int, index int, size int, schema string, table string, filters string,
	results chan<- string, wg *sync.WaitGroup) (*http.Client, *http.Request) {

	api := fmt.Sprintf("http://%s:%d/agent/atomic/detection", host, port)

	defer func() {
		wg.Done()
		if err := recover(); err != nil {
			fmt.Printf("request [%s] error : %v\n", api, err)
			return
		}
	}()

	//------------------

	// 获取分段的数据, 按照固定长度获取 数据，假设固定长度 为默认的 500，则一批次 1-501 500-10001 ...
	sqlString := fmt.Sprintf(`SELECT * FROM (SELECT t.*, ROWNUM AS row_number FROM %s.%s t WHERE ROWNUM <= %d) WHERE row_number >= %d`,
		schema, table, (index+1)*size, 1+index*size)
	result, _ := GlobalDatabaseOracleCache.QueryCombine("", filters, sqlString)

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

func OracleDataRequestByRiver(host string, port int, index int, size int, schema string, table string, filters string,
	results chan<- string, wg *sync.WaitGroup) (*http.Client, *http.Request) {

	api := fmt.Sprintf("http://%s:%d/predict", host, port)

	defer func() {
		wg.Done()
		if err := recover(); err != nil {
			fmt.Printf("request [%s] error : %v\n", api, err)
			return
		}
	}()

	//------------------

	// 获取分段的数据, 按照固定长度获取 数据，假设固定长度 为默认的 500，则一批次 1-501 500-10001 ...
	sqlString := fmt.Sprintf(`SELECT * FROM (SELECT t.*, ROWNUM AS row_number FROM %s.%s t WHERE ROWNUM <= %d) WHERE row_number >= %d`,
		schema, table, (index+1)*size, 1+index*size)
	result, _ := GlobalDatabaseOracleCache.QueryCombineByRiver("", filters, sqlString)
	//fmt.Println(result)
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
		//log.Println(string(body))
		//fmt.Println(pkg.ToJSONObject(string(body)))
		results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}

	return client, req
}

func main() {

	version := flag.Bool("v", false, "Prints current tool version")

	remote := flag.Bool("remote", false, "remote connect or not")
	engine := flag.Int("engine", 0, "detect engine type, 0=activeio, 1=ruishu")

	timeout := flag.Int("timeout", 5, "connection timeout")
	thread := flag.Int("thread", 6, "the thread")

	host := flag.String("host", "", "database host")
	port := flag.String("port", "", "database port")
	username := flag.String("username", "", "database user name")
	password := flag.String("password", "", "database user password")
	sid := flag.String("sid", "", "database sid")

	structs := flag.Bool("structs", false, "direct query database (select)") // 结构检测
	detect := flag.Bool("detect", false, "detect database data (select)")    // 加密检测
	entropy := flag.Bool("entropy", false, "detect entropy data (select)")   // 熵值计算

	filters := flag.String("filter", pkg.OracleStructFilter, "the filter string")
	include := flag.String("include", "", "the include string")
	exclude := flag.String("exclude", "", "the exclude string")
	querySQL := flag.String("query", pkg.OracleStructSql, "the query sql string")

	schema := flag.String("schema", pkg.OracleStructSql, "the schema")
	table := flag.String("table", pkg.OracleStructSql, "the table")
	detected := flag.String("detected", "", "the detected row data")
	detectedPosition := flag.Int("detected-position", 0, "the detected row data position")

	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - ORACLE Database Operator v%s", pkg.AppVersion))
		os.Exit(0)
	}

	start := time.Now()

	connector := &pkg.Connector{}
	connector.Host = *host         //"10.0.0.103"
	connector.Port = *port         //"1521"
	connector.Username = *username //"dong1"
	connector.Password = *password //"root"
	connector.Sid = *sid           //"leora"

	//cstSh, _ := time.LoadLocation("Asia/Shanghai")

	GlobalDatabaseOracleCache.Init("", CreateDatabaseOracle(*remote, *timeout, *connector))

	// 实际内容的解析
	var filterList []map[string]string
	var containList []map[string]string

	if *structs {
		result, _ := GlobalDatabaseOracleCache.Query("", *querySQL)

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
			fmt.Println(fmt.Sprintf("%s.%s", o["SCHEMANAME"], o["TABLENAME"]))
		}

		//fmt.Printf("Finish Tasks Cost=[%v]\n", time.Since(start))
	}

	if *detect {

		count := 0        // 起始位置
		size := 10000 * 2 // 默认抽取数据数

		// TODO 需要控制下最大并发数
		thread := *thread // 线程数

		for {
			// 多个 goroutine 查询数据，每个查询数据中再实现 类似 all promise 的，获取结果
			results := make(chan string)
			var done = false
			var wg sync.WaitGroup

			for i := count * thread; i < (count+1)*thread; i++ {
				wg.Add(1)
				if *engine == 0 {
					go OracleDataRequest("localhost", 56790, i, size, *schema, *table, *exclude, results, &wg)
				} else {
					go OracleDataRequestByRiver("localhost", 56788, i, size, *schema, *table, *exclude, results, &wg)
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
							break
						}
					} else {
						response := pkg.RiverResponse{}
						content := strings.Split(strings.TrimSpace(strings.ReplaceAll(result, "Fetched", "")), "::")
						err := json.Unmarshal([]byte(content[1]), &response) //
						doBreak := false
						index := 0
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
			time.Sleep(10 * time.Millisecond)

			//fmt.Println("run count :", count)
			if done { // 是否存在 分页查询不满，不满认为已经可以结束了
				break
			}

		}

		log.Printf("Finish Tasks Cost=[%v]\n", time.Since(start))
	}

	if *entropy && *engine == 0 {
		//  基础信息获取 - 主键信息
		primaryKeySqlString := fmt.Sprintf("SELECT cu.COLUMN_NAME as pk "+
			"FROM ALL_CONS_COLUMNS cu, ALL_CONSTRAINTS au "+
			"WHERE cu.constraint_name = au.constraint_name"+
			" and au.CONSTRAINT_TYPE = 'P'"+
			" and au.OWNER = '%s'"+
			" and au.TABLE_NAME = '%s'", *schema, *table)
		//fmt.Println(sql)

		result, _ := GlobalDatabaseOracleCache.Query("", primaryKeySqlString)
		primaryKeyAliasSql := ""
		primaryKeySql := ""
		if len(result) > 0 {
			primaryKeyAliasSql = fmt.Sprintf("%s as ADMCID, ", result[0]["PK"])
			primaryKeySql = fmt.Sprintf("%s, ", result[0]["PK"])
		}

		// 计算熵值，IDS 统计计算
		detected := *detected
		detectedColumnValues := strings.Split(detected, "#")
		var columns []string
		var whereCondition []string
		for _, columnNameValue := range detectedColumnValues {
			columnNameValueArray := strings.SplitN(columnNameValue, ":", 2)
			columnName := columnNameValueArray[0]
			columnValue := columnNameValueArray[1]
			if strings.Contains(columnValue, "'") {
				columnValue = strings.Replace(columnValue, "'", "''", -1)
			}
			columns = append(columns, columnName)
			if columnValue == "" {
				whereCondition = append(whereCondition, fmt.Sprintf(`( %s = '' or %s is null )`, columnName, columnName))
			} else {
				//whereCondition = append(whereCondition, strings.Replace(columnNameValue, ":", "='", 1)+"'")
				whereCondition = append(whereCondition, fmt.Sprintf(`%s = '%s'`, columnName, columnValue))
			}
		}

		sql := fmt.Sprintf(`select tt.row_number
from (select ROWNUM AS row_number, %s from %s.%s t) tt
where %s`, strings.Join(columns, ","), *schema, *table, strings.Join(whereCondition, " and "))

		result, _ = GlobalDatabaseOracleCache.Query("", sql)

		if len(result) <= 0 {
			fmt.Println(2)
			fmt.Println("Data detection does not exist")
			return
		}

		breakPoint, _ := strconv.Atoi(result[0]["ROW_NUMBER"]) // TODO 判断位置选项
		breakPointStart := breakPoint - 250 + 1
		breakPointEnd := breakPoint + 250

		if breakPointStart < 0 {
			breakPointStart = 1
			breakPointEnd = 500
		}

		sql = fmt.Sprintf(`SELECT %s %s as combine
FROM (SELECT %s %s, ROWNUM AS row_number FROM %s.%s t WHERE ROWNUM <= %d)
WHERE row_number >= %d`, primaryKeyAliasSql, strings.Join(columns, "||' '||"), primaryKeySql, strings.Join(columns, ","), *schema, *table,
			breakPointEnd, breakPointStart)
		//fmt.Println(sql)

		// 循环 计算 实际熵值与对应 id
		var entropy []string
		var ids []string
		result, _ = GlobalDatabaseOracleCache.Query("", sql)
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
		fmt.Println(500)             //rowCount
		fmt.Println(breakPointStart) //rowStart
		fmt.Println(breakPointEnd)   //rowEnd
		fmt.Println(detected)
	}

	if *entropy && *engine == 1 {
		//  基础信息获取 - 主键信息
		primaryKeySqlString := fmt.Sprintf("SELECT cu.COLUMN_NAME as pk "+
			"FROM ALL_CONS_COLUMNS cu, ALL_CONSTRAINTS au "+
			"WHERE cu.constraint_name = au.constraint_name"+
			" and au.CONSTRAINT_TYPE = 'P'"+
			" and au.OWNER = '%s'"+
			" and au.TABLE_NAME = '%s'", *schema, *table)
		//fmt.Println(sql)

		result, _ := GlobalDatabaseOracleCache.Query("", primaryKeySqlString)
		primaryKeyAliasSql := ""
		primaryKeySql := ""
		if len(result) > 0 {
			primaryKeyAliasSql = fmt.Sprintf("%s as ADMCID, ", result[0]["PK"])
			primaryKeySql = fmt.Sprintf("%s, ", result[0]["PK"])
		}

		//  基础信息获取 - 列名信息
		columnSqlString := fmt.Sprintf("SELECT Listagg(column_name, ',') WITHIN GROUP (ORDER BY column_name) as columns "+
			"FROM ALL_TAB_COLUMNS "+
			"where DATA_TYPE like '%%CHAR%%' "+
			" and OWNER = '%s'"+
			" and TABLE_NAME = '%s'", *schema, *table)
		//fmt.Println(columnSqlString)

		result, _ = GlobalDatabaseOracleCache.Query("", columnSqlString)
		columnKeyAliasSql := ""
		columnSql := ""
		if len(result) > 0 {
			columnKeyAliasSql = fmt.Sprintf("%s as COMBINE", result[0]["COLUMNS"])
			columnSql = fmt.Sprintf("%s ", result[0]["COLUMNS"])
		}

		// 计算熵值，IDS 统计计算
		detected := *detectedPosition
		var one_result string

		sql := fmt.Sprintf(`select %s
from (select ROWNUM AS row_number, t.* from %s.%s t)
where row_number = %d`, columnSql, *schema, *table, detected)

		result, _ = GlobalDatabaseOracleCache.Query("", sql)

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

		// TODO 重新获取检测结果的值
		//detected = "dsadasd"

		breakPoint, _ := strconv.Atoi(result[0]["ROW_NUMBER"]) // TODO 判断位置选项
		breakPointStart := breakPoint - 250 + 1
		breakPointEnd := breakPoint + 250

		if breakPointStart < 0 {
			breakPointStart = 1
			breakPointEnd = 500
		}

		sql = fmt.Sprintf(`SELECT %s %s
FROM (SELECT %s %s, ROWNUM AS row_number FROM %s.%s t WHERE ROWNUM <= %d) tt
WHERE row_number >= %d`, primaryKeyAliasSql, strings.Replace(columnKeyAliasSql, ",", "| |", -1), primaryKeySql, columnSql, *schema, *table,
			breakPointEnd, breakPointStart)

		log.Println(sql)

		// 循环 计算 实际熵值与对应 id
		var entropy []string
		var ids []string
		result, _ = GlobalDatabaseOracleCache.Query("", sql)
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
		fmt.Println(500)             //rowCount
		fmt.Println(breakPointStart) //rowStart
		fmt.Println(breakPointEnd)   //rowEnd
		//fmt.Println(detected)
		fmt.Println(one_result)
	}

	GlobalDatabaseOracleCache.Database.Close()

}
