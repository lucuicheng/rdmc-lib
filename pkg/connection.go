package pkg

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Detection struct {
	Detected  string `json:"detected"`
	Timestamp string `json:"timestamp"`
}

type Response struct {
	Body Detection `json:"body"`
	Code int
	Msg  string
}

type RiverResponse struct {
	Code          int    `json:"code"`
	PredictResult []int  `json:"predict_result"`
	Message       string `json:"message"`
}

type Connector struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
	Sid      string
	Type     string
}

type Connection struct {
	HOST      string
	PORT      string
	DATABASE  string
	USERNAME  string
	PASSWORD  string
	CHARSET   string
	PARSETIME string
	SID       string
}

const (
	OracleStructSql    = "select OWNER as schemaName,table_name as tableName from dba_tables"
	MySQLStructSql     = "SELECT table_schema as SCHEMANAME,table_name as TABLENAME FROM information_schema.tables"
	SqlServerStructSql = `SELECT s.name AS SCHEMANAME, t.name AS TABLENAME FROM sys.schemas AS s JOIN sys.tables AS t ON t.schema_id = s.schema_id order by SCHEMANAME`
)

const (
	OracleStructFilter    = "SYS,SYSTEM,LBACSYS,DBSNMP,XDB,ORDDATA,MDSYS,CTXSYS,DVSYS,WMSYS,GSMADMIN_INTERNAL,OUTLN,APPQOSSYS,OLAPSYS,OJVMSYS,AUDSYS,APEX_040200,ORDSYS,DBSFWUSER"
	MySQLStructFilter     = "sys,information_schema,performance_schema,mysql"
	SqlServerStructFilter = "ReportServer,DWConfiguration,DWQueue,ReportServerTempDB,DWDiagnostics"
)

func FilterValue(val string, args string) bool {
	for _, arg := range strings.Split(args, ",") {
		if val == strings.TrimSpace(arg) {
			return false
		}
	}
	return true
}

func ContainValue(val string, args string) bool {
	for _, arg := range strings.Split(args, ",") {
		if val == strings.TrimSpace(arg) {
			return true
		}
	}
	return false
}

func DatabaseRequest(host string, port int, paramsBytes []byte, results chan<- string, wg *sync.WaitGroup) (*http.Client, *http.Request) {

	api := fmt.Sprintf("http://%s:%d/agent/atomic/detection", host, port)

	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("request [%s] error : %v\n", api, err)
			return
		}
	}()

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

	req, err := http.NewRequest(http.MethodPost, uri.String(), strings.NewReader(string(paramsBytes)))
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
		results <- fmt.Sprintf("Fetched %s", string(body))
	}

	return client, req
}

func Shannon(value string) (bits int) {
	frq := make(map[rune]float64)

	//get frequency of characters
	for _, i := range value {
		frq[i]++
	}

	var sum float64

	for _, v := range frq {
		f := v / float64(len(value))
		sum += f * math.Log2(f)
	}

	bits = int(math.Ceil(sum*-1)) * len(value)
	return
}

type Queue struct {
	items []int
	lock  sync.Mutex
	cond  *sync.Cond
}

func NewQueue() *Queue {
	queue := &Queue{}
	queue.cond = sync.NewCond(&queue.lock)
	return queue
}

func (q *Queue) Enqueue(item int) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.items = append(q.items, item)
	q.cond.Signal()
}

func (q *Queue) Dequeue() (int, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	for len(q.items) == 0 {
		q.cond.Wait()
	}

	item := q.items[0]
	q.items = q.items[1:]

	return item, nil
}

func (q *Queue) IsEmpty() bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.items) == 0
}

func (q *Queue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return len(q.items)
}
