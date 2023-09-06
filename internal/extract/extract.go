package extract

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/cockroachdb/pebble"
	"github.com/deckarep/golang-set"
	"github.com/patrickmn/go-cache"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"rdmc/internal/netdisk"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	TimeDisplay = "2006-01-02 15:04:05 -0700"
)

var count int64
var filteredCount int64
var atomicCount int64
var atomicCountAll int64

type Metadata struct {
	Id     int       `json:"id"`
	Access string    `json:"access"`
	Atime  time.Time `json:"atime"`
	Ctime  time.Time `json:"ctime"`
	Mtime  time.Time `json:"mtime"`
	Gid    uint32    `json:"gid"`
	Uid    uint32    `json:"uid"`
	Size   int64     `json:"size"`
	Inode  uint64    `json:"inode"`
	Suid   bool      `json:"suid"`
	Sgid   bool      `json:"sgid"`
	Md5Sum string    `json:"md5Sum"`
	Head   string    `json:"head"`
	Tail   string    `json:"tail"`
}

func StatMetadata(filename string) string {
	metadata := new(Metadata)

	fileInfo, err := os.Stat(filename)
	if err != nil {
		log.Fatal(err)
	}

	// 获取修改时间
	modTime := fileInfo.ModTime()
	metadata.Mtime = modTime
	//fmt.Println("Modification Time:", modTime)

	// 获取访问时间
	//accessTime := fileInfo.Sys().(*syscall.Stat_t).Atim
	//accessTimeUnix := time.Unix(int64(accessTime.Sec), int64(accessTime.Nsec))
	//metadata.Atime = accessTimeUnix
	//fmt.Println("Access Time:", accessTimeUnix)
	//
	//// 获取创建时间
	//createTime := fileInfo.Sys().(*syscall.Stat_t).Ctim
	//createTimeUnix := time.Unix(int64(createTime.Sec), int64(createTime.Nsec))
	//metadata.Ctime = createTimeUnix
	//fmt.Println("Creation Time:", createTimeUnix)

	// 获取所属用户ID（UID）和所属组ID（GID）
	stat := fileInfo.Sys().(*syscall.Stat_t)
	uid := stat.Uid
	gid := stat.Gid
	inode := stat.Ino
	//fmt.Println("UID:", uid)
	metadata.Uid = uid
	//fmt.Println("GID:", gid)
	metadata.Gid = gid
	//fmt.Println("INO:", inode)
	metadata.Inode = inode

	// 其他元数据
	size := fileInfo.Size()
	//isDir := fileInfo.IsDir()
	mode := fileInfo.Mode()
	hasSUID := mode&os.ModeSetuid != 0
	hasSGID := mode&os.ModeSetgid != 0

	//permissions := fmt.Sprintf("%s %04o", mode, mode.Perm())
	metadata.Access = fmt.Sprintf("%s %04o", mode, mode.Perm())
	metadata.Suid = hasSUID
	metadata.Sgid = hasSGID

	//fmt.Println("Mode:", permissions)
	//fmt.Println("Size:", size)
	metadata.Size = size
	//fmt.Println("Is Directory:", isDir)

	// TODO 优化读取方式与内容
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Failed to open file %s: %v\n", filename, err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Printf("Failed to calculate MD5 for file %s: %v\n", filename, err)
	}

	md5Sum := fmt.Sprintf("%x", hash.Sum(nil))
	metadata.Md5Sum = md5Sum

	metadataBytes, _ := json.Marshal(metadata)
	return string(metadataBytes)
	//fmt.Printf("%v\n", string(metadataBytes))
}

type DatabaseKeyValueCache struct {
	Database *pebble.DB
	Cache    *cache.Cache
}

var GlobalDatabaseKeyValueCache = &DatabaseKeyValueCache{}

func (c *DatabaseKeyValueCache) Init(extra string, databaseInstance *pebble.DB) *DatabaseKeyValueCache {
	c.Cache = cache.New(24*time.Hour, 2*24*time.Hour)

	// 使用系统日志
	//log := logger.SystemAppender().
	//	WithField("component", "SYSTEM").
	//	WithField("category", "DATABASE")

	now := time.Now()
	prefix := now.Format("2006-01-02")

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "pebble", prefix)
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes

	// Set the value of the key "foo" to "bar", with the default expiration time
	cached, found := c.Cache.Get(key)
	if found {
		database := cached.(*pebble.DB)
		//log.Infof("database postgresql [%s] mem pointer is %s\n", prefix, &database)
		c.Database = database
		//return db, nil
	}

	//log.Infof("cached logger not found, start to create new logger[%s] and set it to cache", key)

	c.Cache.Set(key, databaseInstance, cache.NoExpiration) // 设置一个不会过期的数据缓存对象
	c.Database = databaseInstance

	return c
}

func (c *DatabaseKeyValueCache) GetObject(task string) *DatabaseKeyValueCache {

	now := time.Now()
	prefix := now.Format("2006-01-02")

	extra := ""

	key := fmt.Sprintf("global-cache-%s%s-%s", extra, "pebble", prefix)

	//log := logger.SystemAppender().
	//	WithField("component", "SYSTEM").
	//	WithField("category", "KV-DATABASE")

	cached, found := c.Cache.Get(key)
	if found {
		//fmt.Println("use cached postgresql db")
		database := cached.(*pebble.DB)
		//log.Infof("already create one rocksdb %v", database)
		//log.Info("found，database postgresql [%s] mem pointer is %v", prefix, &database)
		c.Database = database
		//return db, nil
	} else {
		// 重新初始化
		//fmt.Println("recreate key-value db")
		//log.Infof("recreate key-value db")
		c.Cache.Flush() // 先清空，再重新初始化，确保只有一个有效缓存
		return c.Init("", c.CreateDatabaseKeyValue(task))
	}

	return c
}

func (c *DatabaseKeyValueCache) CreateDatabaseKeyValue(task string) *pebble.DB {

	// 注册到本地
	prefix := task
	kvSourceStorePath := "/opt/frs/" + prefix // 需要创建单独的数据
	kvSourceDB, err := pebble.Open(strings.Join([]string{kvSourceStorePath}, ""), &pebble.Options{FormatMajorVersion: 8})
	if err != nil {
		log.Printf("db open failed, err : %v\n", err)
	}
	return kvSourceDB
}

func (c *DatabaseKeyValueCache) Save(items []string) {
	// TODO SEND to request
	kvSourceDB := c.Database

	kvBatch := kvSourceDB.NewIndexedBatch()

	for _, line := range items {
		lineParts := strings.Split(line, "::")

		key := lineParts[0]
		value := lineParts[1]

		if err := kvBatch.Set([]byte(key), []byte(value), nil); err != nil {
			continue
		}
	}

	kvBatch.Commit(pebble.Sync)
	kvBatch.Close()
}

type BatchQueue struct {
	queue     []string
	batchSize int
	mutex     sync.Mutex
}

func NewBatchQueue(batchSize int) *BatchQueue {
	return &BatchQueue{
		queue:     make([]string, 0),
		batchSize: batchSize,
	}
}

func (q *BatchQueue) Enqueue(item string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.queue = append(q.queue, item)
	if len(q.queue) >= q.batchSize {
		q.flush()
	}
}

func (q *BatchQueue) DequeueAll() []string {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	items := make([]string, len(q.queue))
	copy(items, q.queue)
	q.queue = make([]string, 0)
	return items
}

func (q *BatchQueue) flush() {
	// 执行批量操作的逻辑
	//fmt.Println("Batch processing:", len(q.queue))

	GlobalDatabaseKeyValueCache.Save(q.queue)
	//OracleDataRequest("10.0.0.13", 56790, q.queue)

	// 清空队列
	q.queue = make([]string, 0)
}

type Duplicate struct {
	FilePath string
	Mtime    time.Time
	Tag      string
	Count    int64
}

type Target struct {
	Path     string
	Metadata Metadata
}

type Output struct {
	Name  string `json:"name"`
	Mtime string `json:"mtime"`
}

func keyUpperBound(b []byte) []byte {
	end := make([]byte, len(b))
	copy(end, b)
	for i := len(end) - 1; i >= 0; i-- {
		end[i] = end[i] + 1
		if end[i] != 0 {
			return end[:i+1]
		}
	}
	return nil // no upper-bound
}

func prefixIterOptions(prefix []byte) *pebble.IterOptions {
	return &pebble.IterOptions{
		//TableFilter: func(userProps map[string]string) bool {
		//	fmt.Println(userProps["name"])
		//	return false
		//},
		//RangeKeyMasking: pebble.RangeKeyMasking{},
		//KeyTypes: pebble.IterKeyTypePointsAndRanges,

		LowerBound: prefix,
		UpperBound: keyUpperBound(prefix),
	}
}

func filter(target Target, handle func(target Target) (bool, string), results chan<- Duplicate, num chan<- int64) bool {

	//defer wg.Done()
	if filtered, tag := handle(target); filtered {
		atomic.AddInt64(&filteredCount, 1)
		results <- Duplicate{FilePath: target.Path, Mtime: target.Metadata.Mtime, Tag: tag, Count: filteredCount}
	}

	num <- 1

	return true
}

var supportExtList = []string{
	".docx",
	".xlsx",
	".pptx",
	".pdf",
	".zip",
	".tar.gz",
	".xml",
}

type PDF struct {
	Head int `json:"head"`
	Tail int `json:"tail"`
}

func (p *PDF) check(path string) (bool, bool, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	var headSize = 10
	if p.Head != 0 {
		headSize = p.Head
	}
	var tailSize = 512
	if p.Tail != 0 {
		tailSize = p.Tail
	}

	// Read the first 10 bytes
	head := make([]byte, headSize)
	_, err = io.ReadFull(file, head)
	if err != nil {
		return false, false, err
	}

	// Print the head info
	//fmt.Printf("HEADER : %s\n", strconv.Quote(string(head)))

	// Check that the file is a PDF
	if strings.Contains(strconv.Quote(string(head)), "PDF") {

		// Read the file trailer
		file.Seek(int64(0-tailSize), io.SeekEnd)
		trailer := make([]byte, tailSize)
		_, err = io.ReadFull(file, trailer)
		if err != nil {
			return true, false, err
		}

		tail := string(trailer)

		if strings.Contains(tail, "Encrypt") {
			//fmt.Println("pdf is Encrypted")
			return true, true, err
		}

		return true, false, err
	}

	return false, false, err

	//// Find the start of the trailer dictionary
	//start := len(trailer) - 2
	//for trailer[start] != '<' {
	//	//fmt.Println(start, strconv.Quote(string(trailer[start])))
	//	start--
	//}
}

type XML struct {
	Head int `json:"head"`
	Tail int `json:"tail"`
}

func (x *XML) check(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Create a byte slice to read the file's head
	const headSize = 512
	head := make([]byte, headSize)

	// Read the file's head into the byte slice
	_, err = file.Read(head)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Check if the head contains an XML declaration or root element opening tag
	var xmlDeclaration struct {
		XMLName xml.Name `xml:"xml"`
	}
	err = xml.Unmarshal(head, &xmlDeclaration)
	if err == nil {
		return true, nil
	}

	var rootElement struct {
		XMLName xml.Name
	}
	err = xml.Unmarshal(head, &rootElement)
	if err == nil {
		return true, nil
	}

	return false, nil
}

type GZIP struct {
	Head int `json:"head"`
	Tail int `json:"tail"`
}

func (g *GZIP) check(path string) (bool, bool, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	var headSize = 10
	if g.Head != 0 {
		headSize = g.Head
	}
	//var tailSize = 512
	//if g.Tail != 0 {
	//	tailSize = g.Tail
	//}

	// Read the first 10 bytes
	head := make([]byte, headSize)
	_, err = io.ReadFull(file, head)
	if err != nil {
		return false, false, err
	}

	/**
	\x1f\x8b\b\x00\x00\x00\x00\x00\x00\x03\xec=\xdbr\xe36\x96y\xe6W`\xed\xadr\x
	Salted__(*\xa0\xaa\x0f\x15\xf7&\x1e\xc4S\xd1\xf6\xc3\xd8\xf6r$Q@S\xa4\x9a\t\xaa\xdc\xd5-g\xce\xf9E\xf5P.=\x1e\xc6iB\xe2\xe8\x93v\x1d\xa6N\xa4\x9ai!\xdc[\x14\x8bQ
	Salted__\x92C\xdch\xa070\xe7\v\x96\x81\u0096\xe2\x10\xca\x18\x87 \xc9ѫ\x1e}\xd8\xed.%\xb27y3\xd6\xf7\x9c\xd2en5e+h\xe5\xdd{/\xe1\x05J\xb5Q\x14\xb18\xb3@
	Salted__\xbb\xc8Qs\xa75\xafPqto\xd1\xf580\x9f6\xa5\x0eJ\x93\U0007e05bƘĬ\x1b{\x9fՂFd\xe7\x12\x18\x7f\x12\xa2_\xe2\x90\xc3\"\xaa\x94O\xad}+\x15\xaf\x85
	*/
	// Print the head info
	fmt.Printf("GZIP HEADER : %s\n", strconv.Quote(string(head)))

	// Check that the file is a PDF
	if strings.Contains(strconv.Quote(string(head)), "\\x1f\\x8b\\b\\x00") {
		return true, false, err
	} else if strings.Contains(strconv.Quote(string(head)), "PK\\x03\\x04\\x14\\x00\\x01\\x00\\b\\x00") ||
		strings.Contains(string(head), "Salted__") {
		return true, true, err
	}

	return false, false, err
}

type ZIP struct {
	Head int `json:"head"`
	Tail int `json:"tail"`
}

func (z *ZIP) check(path string) (bool, bool, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	var headSize = 10
	if z.Head != 0 {
		headSize = z.Head
	}
	//var tailSize = 512
	//if z.Tail != 0 {
	//	tailSize = z.Tail
	//}

	// Read the first 10 bytes
	head := make([]byte, headSize)
	_, err = io.ReadFull(file, head)
	if err != nil {
		return false, false, err
	}

	// Print the head info
	//fmt.Printf("ZIP HEADER : %s\n", strconv.Quote(string(head)))

	// PK\x03\x04\x14\x00\x00\x00\b\x00\xecX  不加密
	// PK\x03\x04\x14\x00\x00\x00\b\x00
	// PK\x03\x04\x14\x00\t\x00\b\x00np\x12Uo;\xa6\xaa\xef\xb3\a\x00  加密
	// PK\x03\x04\x14\x00\t\x00\b\x00nM\xf5V\x0f\xf0Ǖ\xe4\x01\x00\x00\xe4\x02\x00\x00\x16\x00\x1c\x00co
	// PK\x03\x04\x14\x00\x00\x00\b\x00
	// PK\x03\x04\x14\x00\x01\x00\b\x00

	// Check that the file is a PDF
	if strings.Contains(strconv.Quote(string(head)), "PK\\x03\\x04\\x14\\x00\\x00\\x00\\b\\x00") {
		return true, false, err
	} else if strings.Contains(strconv.Quote(string(head)), "PK\\x03\\x04\\x14\\x00\\x01\\x00\\b\\x00") ||
		strings.Contains(strconv.Quote(string(head)), "PK\\x03\\x04\\x14\\x00\\t\\x00\\b\\x00") {
		return true, true, err
	}

	return false, false, err
}

type Office struct {
	Name      string `json:"path"`
	Extension string `json:"extension"`
	Head      int    `json:"head"`
	Tail      int    `json:"tail"`
}

type OfficeType string

var a = "PK\\x03\\x04\\x14\\b\\xa9T\\xf3V\\xadR\\xa5\\x91\\x95\\x01\\xca\\x06\\x13[Content_Types].xml"

const (
	OfficeWord       OfficeType = "PK\\x03\\x04\\x14\\x06\\b!ߤ\\xd2lZ\\x01 \\x05\\x13\\b\\x02[Content_Types].xml"
	OfficeExcel      OfficeType = "PK\\x03\\x04\\x14\\x06\\b!A7\\x82\\xcfn\\x01\\x04\\x05\\x13\\b\\x02[Content_Types].xml"
	OfficePowerPoint OfficeType = "PK\\x03\\x04\\x14\\x06\\b!\\xdf\\xcc\\x18\\xf5\\xad\\x01F\\f\\x13\\b\\x02[Content_Types].xml"

	OfficeDocument     OfficeType = "PK\\x03\\x04\\n\\x87N\\xe2@\\tdocProps/"
	OfficeSpreadsheet  OfficeType = "PK\\x03\\x04\\n\\x87N\\xe2@\\tdocProps/"
	OfficePresentation OfficeType = "PK\\x03\\x04\\n\\x87N\\xe2@\\x04ppt/"

	OfficePythonDoc    OfficeType = "\\xe3V\\xadR\\xa5\\x91\\x95\\x01\\xca\\x06\\x13[Content_Types].xml"
	OfficePythonSheet  OfficeType = "\\xe3VFZ\\xc1\\f\\x82\\xb1\\x10docProps/app.xml"
	OfficePythonSlider OfficeType = "\\xe3VrΪ\\xe7\\xbd\\x01<\\r\\x13[Content_Types].xml"
)

func (x *Office) check(path string, tag string, officeTypes ...OfficeType) (bool, bool, error) {

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	var headSize = 64
	if x.Head != 0 {
		headSize = x.Head
	}
	var tailSize = 960
	if x.Tail != 0 {
		tailSize = x.Tail
	}

	// 设置读取的起始位置
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		//fmt.Println("Failed to seek file:", err)
		return false, false, err
	}

	// 读取指定部分头信息
	buffer := make([]byte, headSize) // 设置读取的长度
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		//fmt.Println("Failed to read file:", err)
		return false, false, err
	}

	// 检查是否达到读取的终止位置
	if n < headSize {
		//fmt.Println("End of file reached before reading desired range")
		return false, false, err
	}

	headInfo := strconv.Quote(string(buffer[:n]))
	headInfo = strings.Replace(headInfo, "\\x00", "", -1)
	//fmt.Printf("HEADER : %s\n", headInfo)

	// 读取文件指定部分尾部信息
	file.Seek(int64(0-tailSize), io.SeekEnd)
	trailer := make([]byte, tailSize)
	_, err = io.ReadFull(file, trailer)
	if err != nil {
		return false, false, err
	}

	tail := string(trailer)
	tail = strings.Replace(strconv.Quote(string(trailer)), "\\x00", "", -1)
	//fmt.Printf("TAIL : %s\n", tail)

	for _, officeType := range officeTypes {

		if strings.Contains(headInfo, string(officeType)) || strings.Contains(tail, tag) {

			if strings.Contains(tail, "Encrypt") {
				//fmt.Println("pdf is Encrypted")
				return true, true, err
			}

			return true, false, err
		} else {

			headSize = 560

			// 设置读取的起始位置
			_, err = file.Seek(1024, io.SeekStart)
			if err != nil {
				//fmt.Println("Failed to seek file:", err)
				return false, false, err
			}

			// 读取指定部分头信息
			buffer := make([]byte, headSize) // 设置读取的长度
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				//fmt.Println("Failed to read file:", err)
				return false, false, err
			}

			// 检查是否达到读取的终止位置
			if n < headSize {
				//fmt.Println("End of file reached before reading desired range")
				return false, false, err
			}

			headInfo := strconv.Quote(string(buffer[:n]))
			headInfo = strings.Replace(headInfo, "\\x00", "", -1)
			headInfo = strings.Replace(headInfo, "\\xff", "", -1)
			//fmt.Printf("HEADER : %s\n", headInfo)

			if strings.Contains(headInfo, "EncryptedPackage") {
				//fmt.Println("pdf is Encrypted")
				return true, true, err
			}

		}
	}

	return false, false, err
}

func check(path string) (bool, string, error) {
	ext := filepath.Ext(path)

	// ---------------------------------------------------------------------------
	tag := "unmatched-suffix"
	// TODO 可优化成并发查询, 根据后缀优先调用具体 office 检测

	office := Office{Head: 512, Tail: 960}
	isOffice, isEncrypted, err := office.check(path, "word/document.xml", OfficeWord, OfficeDocument, OfficePythonDoc)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isOffice {
		//fmt.Printf("is Word/Document? %v and pdf is encrypted? %v\n", isOffice, isEncrypted)
		tag := "matched-suffix-docx"
		if ext != ".docx" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".docx" {
			tag = "unmatched-suffix-docx"
		}
	}

	office = Office{Head: 512, Tail: 512}
	isOffice, isEncrypted, err = office.check(path, "worksheets/sheet1.xml", OfficeExcel, OfficeSpreadsheet, OfficePythonSheet)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isOffice {
		//fmt.Printf("is Excel/Spreadsheet? %v and pdf is encrypted? %v\n", isOffice, isEncrypted)
		tag := "matched-suffix-xlsx"
		if ext != ".xlsx" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".xlsx" {
			tag = "unmatched-suffix-xlsx"
		}
	}

	office = Office{Head: 512, Tail: 512}
	isOffice, isEncrypted, err = office.check(path, "ppt/slides/slide1.xml", OfficePowerPoint, OfficePresentation, OfficePythonSlider)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isOffice {
		//fmt.Printf("is PowerPoint/Presentation? %v and pdf is encrypted? %v\n", isOffice, isEncrypted)
		tag := "matched-suffix-pptx"
		if ext != ".pptx" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".pptx" {
			tag = "unmatched-suffix-pptx"
		}
	}

	// ---------------------------------------------------------------------------

	pdf := PDF{}
	isPDF, isEncrypted, err := pdf.check(path)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isPDF {
		//fmt.Printf("is pdf? %v, and pdf is encrypted? %v is extension name corect %v\n", isPDF, isEncrypted, ext == ".pdf")
		tag := "matched-suffix-pdf"
		if ext != ".pdf" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".pdf" {
			tag = "unmatched-suffix-pdf"
		}
	}

	// ---------------------------------------------------------------------------

	zip := ZIP{Head: 32}
	isZIP, isEncrypted, err := zip.check(path)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isZIP {
		//fmt.Printf("is ZIP? %v, and zip is encrypted? %v, is extension name corect %v\n", isZIP, isEncrypted, ext == ".zip")
		tag := "matched-suffix-zip"
		if ext != ".zip" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".zip" {
			tag = "unmatched-suffix-zip"
		}
	}

	// ---------------------------------------------------------------------------

	gzip := GZIP{}
	isGZIP, isEncrypted, err := gzip.check(path)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isGZIP {
		//fmt.Printf("is GZIP? %v, and gzip is encrypted? %v\n", isGZIP, isEncrypted)
		tag := "matched-suffix-gzip"
		if ext != ".gz" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".gz" {
			tag = "unmatched-suffix-gzip"
		}
	}

	// ---------------------------------------------------------------------------

	xml := XML{}
	isXML, err := xml.check(path)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isXML {
		//fmt.Printf("is XML? %v\n", isXML)
		tag := "matched-suffix-xml"
		if ext != ".xml" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".xml" {
			tag = "unmatched-suffix-xml"
		}
	}

	return isEncrypted, tag, nil
}

func generateReportContent(path string, result map[string][]Output, alias string, threshold int) {
	// 最后写入 目标文件
	// 创建文件，如果文件已存在则会覆盖
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("file create failed : %v\n", err)
		defer file.Close()
	}
	defer file.Close()

	// 创建一个写入缓冲区
	writer := bufio.NewWriter(file)

	// 输出重复的 MD5 值及其数量  写入文件的每一行内容
	for key, paths := range result {
		keys := strings.Split(key, ":")
		tag := keys[0]
		value := keys[1]

		//RANSOMWARE
		if len(paths) > threshold {
			_, err = writer.WriteString(fmt.Sprintf("%s: %s", alias, value))
			if tag != "" {
				_, err = writer.WriteString(fmt.Sprintf("(%s)", tag))
			}
			if len(paths) > 0 {
				_, err = writer.WriteString(fmt.Sprintf(", Count: %d", len(paths)))
				bytes, _ := json.Marshal(paths)
				_, err = writer.WriteString(fmt.Sprintf(", Detail: %s", string(bytes)))
			}
			_, err = writer.WriteString(fmt.Sprintf("\r\n"))
		}

		//_, err := writer.WriteString(fmt.Sprintf("%s: %s, Count: %d\r\n", alias, value, len(paths)))
		// 循环写入具体路径
		//for _, path := range paths {
		//	writer.WriteString(path + "\r\n")
		//}
		//writer.WriteString("\r\n")
		if err != nil {
			log.Printf("data write failed : %v\n", err)
		}
	}

	// 刷新缓冲区，确保所有数据都写入文件
	err = writer.Flush()
	if err != nil {
		log.Printf("file flush failed : %v\n", err)
	}
}

func countAndPositions(s string) (int, bool, bool, bool) {
	count := 0
	first := false
	last := false
	middle := false

	for i, char := range s {
		if char == '*' {
			count++
			if i == 0 {
				first = true
			} else if i == len(s)-1 {
				last = true
			} else {
				middle = true
			}
		}
	}

	return count, first, middle, last
}

func matchSuffix(fileName string, target string) bool {

	_, isAtStart, isAtEnd, isAtMiddle := countAndPositions(target)

	//fmt.Println(isAtStart, isAtEnd, isAtMiddle)

	if isAtStart && !isAtEnd && !isAtMiddle { //通配符 只在 首位
		//fmt.Println("start", target)
		return strings.HasSuffix(fileName, target[1:])
	} else if !isAtStart && !isAtEnd && isAtMiddle { // 通配符只在 末位
		//fmt.Println(target, "end")
		return strings.HasPrefix(fileName, target[:len(target)-1])
	} else if !isAtStart && !isAtEnd && !isAtMiddle { // 通配符不存在
		//fmt.Println(target, "noe")
		return fileName == target
	} else {
		matched := true
		parts := strings.Split(target, "*")

		lastIndex := len(parts) - 1
		for i, part := range parts {
			if part != "" {
				matched = strings.Contains(fileName, part)

				if !matched {
					break
				}

				if i == 0 {
					index := strings.Index(fileName, part)
					matched = index == 0
				}

				if i == lastIndex {
					index := strings.Index(fileName, part)
					matchedIndex := len(strings.Replace(fileName, part, "", -1))
					matched = index == matchedIndex
				}
			}
		}

		return matched
	}

	return false
}

func SystemMaxConcurrency() int {
	if runtime.NumCPU() <= 4 {
		return 1
	} else {
		return runtime.NumCPU() / 3 // 使用 三分之一的资源
	}
}

// Origin 抽取原始数据
func Origin(root string, task string, count int64) int64 {

	//start := time.Now()

	// Create a channel to receive file paths
	files := make(chan string)

	// Create a channel to signal when all workers are done
	done := make(chan bool)

	// Define the number of worker goroutines to use
	numWorkers := SystemMaxConcurrency()
	queue := NewBatchQueue(50000)

	log.Printf("cpu core : [%d], numWorkers : [%d], queue size : [%d]\n", runtime.NumCPU(), numWorkers, 50000)

	// Running the workers
	for i := 0; i < numWorkers; i++ {
		go func() {
			for filePath := range files {

				metadata := StatMetadata(filePath)
				queue.Enqueue(fmt.Sprintf("%s::%s", filePath, metadata))

				//log.Println(metadata)

				//ext := filepath.Ext(filePath) // TODO 增加非后缀类型的
				//
				//// TODO 内部再次增加 异步循环，比对多个 extFilter 组
				//for _, filter := range ExtFilter {
				//	if ext == filter[1:] {
				//		fmt.Println(filePath)
				//		//break
				//	}
				//}

				atomic.AddInt64(&count, 1)
			}
			done <- true
		}()
	}

	// Traverse the root directory recursively and send file paths to the channel
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files <- path
		}
		return nil
	})
	if err != nil {
		fmt.Printf("ERROR is %v", err)
	}

	// Close the channel to signal no more files will be sent
	close(files)

	// Wait for all workers to finish
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	items := queue.DequeueAll()
	GlobalDatabaseKeyValueCache.Save(items)

	countNum := atomic.LoadInt64(&count)

	dataMap := make(map[string][]Output)
	dataMap[fmt.Sprintf(`%s:%d`, root, countNum)] = []Output{}
	path := fmt.Sprintf("/opt/frs/%s/report.frs", task)

	// 第一步创建文件
	os.Remove(path)
	file, err := os.Create(path)
	if err != nil {
		log.Printf("file create failed : %v\n", err)
		defer file.Close()
	}
	defer file.Close()

	// 第二步 写追加文件
	generateReportContent(path, dataMap, "FileCount", -1)

	return countNum
}

// Fetch 拉取网盘原始数据
func Fetch(root string, id int, ids []string, task string, count int64) int64 {

	queue := NewBatchQueue(500)

	prefix := fmt.Sprintf("/opt/frs/%s", task)
	netDisk := netdisk.NetDisk{Host: "10.0.0.122", Port: 30179, Username: "admin", Password: "edoc2edoc2"} // TODO 优化动态传入参数

	// TODO 从缓存中获取 token
	if true {
		netDisk = netdisk.RegretNetDiskWithToken(netDisk)
	} else {
		netDisk.Token = ""
	}
	log.Printf("token is %s\n", netDisk.Token)

	// 文件目录下的所有内容获取
	fileAndFolderList := netdisk.FileAndFolderFetch(netDisk, root, id) // 默认同步的路径
	//sizeCopied := make([]int, len(fileAndFolderList.Data.FilesInfo))
	//fetchedCount := len(fileAndFolderList.Data.FilesInfo)
	// TODO 复制原始 list + 从上次检测结果中过滤为缓存和检测的

	// 增减原子应用的 atomic 计数 并发下载
	var wg sync.WaitGroup // 创建一个等待组
	maxConcurrency := 1   // 最大并发打开文件数
	//results := make(chan error)                     // 创建一个通道来接收 重复数据 结果
	fileChan := make(chan netdisk.FileInfo, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for file := range fileChan {
				localDiskFullName := fmt.Sprintf("%s/%s", prefix, file.VirtualFullName)
				//fmt.Printf("downloaded file: %s, id is [%d]\n", file.VirtualFullName, file.FileId)
				atomic.AddInt64(&count, 1)
				//  直接将元数据放入队列
				metadata := Metadata{
					// TODO 构造新的元数据
					Id:     file.FileId,
					Access: file.FileCreateOperatorName + "," + strconv.Itoa(len(file.FilePermissions)),
					Atime:  netdisk.CSTDateTime(file.FileArchiveTime),
					Ctime:  netdisk.CSTDateTime(file.FileCreateTime),
					Mtime:  netdisk.CSTDateTime(file.FileModifyTime),
					Gid:    0,
					Uid:    0,
					Size:   file.FileCurSize,
					Inode:  0,
					Suid:   false,
					Sgid:   false,
					Md5Sum: file.FileMD5,
				} // 构建模拟 元数据
				// TODO 增加 md5 和权限信息查询
				metadataBytes, _ := json.Marshal(metadata)
				queue.Enqueue(fmt.Sprintf("%s::%s", localDiskFullName, string(metadataBytes)))
			}
			wg.Done()
		}()
	}

	go func() {
		for _, file := range fileAndFolderList.Data.FilesInfo {
			//fmt.Println(len(ids), ids, len(ids) == 0 || (len(ids) == 1 && ids[0] == ""))
			if len(ids) > 0 && ids[0] != "" {

				var interfaceSlice []interface{}

				// 将字符串切片转换为空接口切片
				for _, str := range ids {
					interfaceSlice = append(interfaceSlice, str)
				}

				idIntSet := mapset.NewSet(interfaceSlice...)
				if idIntSet.Contains(strconv.Itoa(file.FileId)) {
					fileChan <- file
				}
			} else if len(ids) == 0 || (len(ids) == 1 && ids[0] == "") {
				fileChan <- file
			}
			//fmt.Printf("file: %s, id is [%d], md5 is [%s] and permission is %v\n", file.VirtualFullName, file.FileId, file.FileMD5, file.FilePermissions)
		}
		//for _, folder := range fileAndFolderList.Data.FoldersInfo {
		//	fmt.Printf("folder: %s, id is [%d]\n", folder.VirtualFullName, folder.FolderId)
		//}
		close(fileChan)
	}()

	wg.Wait()

	//fmt.Println("all done")

	//  及时登出 token
	netdisk.ResetNetDiskWithoutToken(netDisk)

	items := queue.DequeueAll()
	GlobalDatabaseKeyValueCache.Save(items)

	countNum := atomic.LoadInt64(&count)

	// 同步采集报告

	dataMap := make(map[string][]Output)
	dataMap[fmt.Sprintf(`%s:%d`, root, countNum)] = []Output{}
	path := fmt.Sprintf("/opt/frs/%s/report.frs", task)

	// 第一步创建文件
	os.Remove(path)
	file, err := os.Create(path)
	if err != nil {
		log.Printf("file create failed : %v\n", err)
		defer file.Close()
	}
	defer file.Close()

	// 第二步 写追加文件
	generateReportContent(path, dataMap, "FileCount", -1)

	return countNum
}

// Download 同步网盘数据到本地
func Download(root string, id int, ids []string, task string, count int64) int {

	prefix := fmt.Sprintf("/opt/frs/%s", task)
	netDisk := netdisk.NetDisk{Host: "10.0.0.122", Port: 30179, Username: "admin", Password: "edoc2edoc2"} // TODO 优化动态传入参数

	// TODO 从缓存中获取 token
	if true {
		netDisk = netdisk.RegretNetDiskWithToken(netDisk)
	} else {
		netDisk.Token = ""
	}
	log.Printf("token is %s\n", netDisk.Token)

	countNum := 0
	maxConcurrency := 2 //SystemMaxConcurrency() // 最大并发打开文件数,下载比较特殊，只能一个个下载

	results := make(chan Duplicate)               // 创建一个通道来接收 过滤数据 结果
	num := make(chan int64)                       // 创建一个通道来接收 过滤数据 计数
	fileChan := make(chan Target, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

	// 启动多个 goroutine 处理文件
	for i := 0; i < maxConcurrency-1; i++ {
		go func() {
			for target := range fileChan {
				filter(target,
					// 自定义的 过滤条件
					func(target Target) (bool, string) {

						downloaded := false
						//fmt.Println(target.Path, target.Metadata)
						virtualFullName := strings.Replace(target.Path, prefix, "", -1)
						err := netdisk.SingleFileDownload(netdisk.NetDiskFile{NetDisk: netDisk, FileId: target.Metadata.Id}, target.Path, 0)
						if err != nil {
							log.Printf("file: %s downloaded failed, id is [%d], %v\n", virtualFullName, target.Metadata.Id, err)
							downloaded = false
							// TODO 队列优化机制， 同样使用递归，直到结束位置
						} else {
							//fmt.Printf("downloaded file: %s, id is [%d]\n", file.VirtualFullName, file.FileId)
							downloaded = true
							//fmt.Printf("downloaded file: %s, id is [%d], md5 is [%s] and permission is %v\n", file.VirtualFullName, file.FileId, file.FileMD5, file.FilePermissions)
						}
						time.Sleep(50 * time.Millisecond)

						return downloaded, "synced" //
					},
					results, num)
			}
		}()
	}

	go func() {
		kvSourceDB := GlobalDatabaseKeyValueCache.Database
		kvBatch := kvSourceDB.NewIndexedBatch()
		// 遍历数据库
		iter, _ := kvBatch.NewIter(nil) // prefixIterOptions([]byte(""))
		for iter.First(); iter.Valid(); iter.Next() {
			// Only keys beginning with "prefix" will be visited.
			key := iter.Key()
			valueBytes, err := iter.ValueAndErr()

			atomic.AddInt64(&atomicCountAll, 1)
			if err != nil {
				//log.Errorf("fetch value bytes %v", err)
				continue
			}
			var metadata Metadata
			json.Unmarshal(valueBytes, &metadata)
			fileChan <- Target{Path: string(key), Metadata: metadata}
		}

		iter.Close()
		kvBatch.Commit(pebble.Sync)
		kvBatch.Close()
	}()

	duplicateMap := make(map[string][]Output)
	done := make(chan string)

	go func() {
		for {
			select {
			case duplicated, _ := <-results:
				//atomic.AddInt64(&resultCount, 1)
				//fmt.Println(fmd5, notFoundPathCount+fmd5.Count, lineCount)
				_, fileName := path.Split(duplicated.FilePath) // 解析路径

				key := duplicated.Tag + ":" + fileName // 组合成 key
				duplicateMap[key] = append(duplicateMap[key], Output{
					Name:  duplicated.FilePath,
					Mtime: duplicated.Mtime.Format(TimeDisplay),
				})
				countNum++

			case n, _ := <-num:
				atomic.AddInt64(&atomicCount, n)
				//547432
				if count == atomicCount {
					done <- "done"
					return
				}
			default:

			}
		}
	}()

	fmt.Println(<-done)

	// TODO 后续可额外记录，未下载成功的数据

	return countNum
}

// Duplicated 抽取重复数据
func Duplicated(task string, count int64) int {
	countNum := 0
	maxConcurrency := SystemMaxConcurrency() // 最大并发打开文件数

	results := make(chan Duplicate)               // 创建一个通道来接收 重复数据 结果
	num := make(chan int64)                       // 创建一个通道来接收 重复数据 结果
	fileChan := make(chan Target, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

	// 启动多个 goroutine 处理文件
	for i := 0; i < maxConcurrency-1; i++ {
		go func() {
			for target := range fileChan {
				filter(target,
					// 自定义的 过滤条件
					func(target Target) (bool, string) {
						return target.Metadata.Size < 1024*1024*1024, fmt.Sprintf("<1M && %s", target.Metadata.Md5Sum) // 需要改成小于 1M
					},
					results, num)
			}
		}()
	}

	go func() {
		kvSourceDB := GlobalDatabaseKeyValueCache.Database
		kvBatch := kvSourceDB.NewIndexedBatch()
		// 遍历数据库
		iter, _ := kvBatch.NewIter(nil) // prefixIterOptions([]byte(""))
		for iter.First(); iter.Valid(); iter.Next() {
			// Only keys beginning with "prefix" will be visited.
			key := iter.Key()
			valueBytes, err := iter.ValueAndErr()

			atomic.AddInt64(&atomicCountAll, 1)
			if err != nil {
				//log.Errorf("fetch value bytes %v", err)
				continue
			}
			var metadata Metadata
			json.Unmarshal(valueBytes, &metadata)
			fileChan <- Target{Path: string(key), Metadata: metadata}
		}

		iter.Close()
		kvBatch.Commit(pebble.Sync)
		kvBatch.Close()
	}()

	duplicateMap := make(map[string][]Output)
	done := make(chan string)

	go func() {
		for {
			select {
			case duplicated, _ := <-results:
				//atomic.AddInt64(&resultCount, 1)
				//fmt.Println(fmd5, notFoundPathCount+fmd5.Count, lineCount)
				_, fileName := path.Split(duplicated.FilePath) // 解析路径

				key := duplicated.Tag + ":" + fileName // 组合成 key
				duplicateMap[key] = append(duplicateMap[key], Output{
					Name:  duplicated.FilePath,
					Mtime: duplicated.Mtime.Format(TimeDisplay),
				})
				if len(duplicateMap[key]) > 9 { // 10 应该为默认的阈值
					countNum++
					// TODO 格式化后写入文件作为报表
					//log.Println(key, duplicateMap[key])
				}

			case n, _ := <-num:
				atomic.AddInt64(&atomicCount, n)
				//547432
				if count == atomicCount {
					done <- "done"
					return
				}
			default:

			}
		}
	}()

	fmt.Println(<-done)

	path := fmt.Sprintf("/opt/frs/%s/report.frs", task)
	generateReportContent(path, duplicateMap, "DUPLICATED", 9)

	return countNum
}

// Permissions 抽取带指定权限的数据
func Permissions(root string, count int64) int {
	countNum := 0
	maxConcurrency := SystemMaxConcurrency() // 最大并发打开文件数

	results := make(chan Duplicate)               // 创建一个通道来接收 重复数据 结果
	num := make(chan int64)                       // 创建一个通道来接收 重复数据 结果
	fileChan := make(chan Target, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

	// 启动多个 goroutine 处理文件
	for i := 0; i < maxConcurrency-1; i++ {
		go func() {
			for target := range fileChan {
				filter(target,
					// 自定义的 过滤条件
					func(target Target) (bool, string) {
						tag := fmt.Sprintf("777=%v suid=%v sgid=%v", strings.Contains(target.Metadata.Access, "777"), target.Metadata.Suid, target.Metadata.Sgid)
						return strings.Contains(target.Metadata.Access, "777") || target.Metadata.Suid || target.Metadata.Sgid, tag
					},
					results, num)
			}
		}()
	}

	go func() {
		kvSourceDB := GlobalDatabaseKeyValueCache.Database
		kvBatch := kvSourceDB.NewIndexedBatch()
		// 遍历数据库
		iter, _ := kvBatch.NewIter(nil) //prefixIterOptions([]byte(""))
		for iter.First(); iter.Valid(); iter.Next() {
			// Only keys beginning with "prefix" will be visited.
			key := iter.Key()
			valueBytes, err := iter.ValueAndErr()

			atomic.AddInt64(&atomicCountAll, 1)
			if err != nil {
				//log.Errorf("fetch value bytes %v", err)
				continue
			}
			var metadata Metadata
			json.Unmarshal(valueBytes, &metadata)
			fileChan <- Target{Path: string(key), Metadata: metadata}
		}

		iter.Close()
		kvBatch.Commit(pebble.Sync)
		kvBatch.Close()
	}()

	duplicateMap := make(map[string][]Output)
	done := make(chan string)

	go func() {
		for {
			select {
			case duplicated, _ := <-results:
				//atomic.AddInt64(&resultCount, 1)
				//fmt.Println(fmd5, notFoundPathCount+fmd5.Count, lineCount)
				_, fileName := path.Split(duplicated.FilePath) // 解析路径

				key := duplicated.Tag + ":" + fileName // 组合成 key
				duplicateMap[key] = append(duplicateMap[key], Output{
					Name:  duplicated.FilePath,
					Mtime: duplicated.Mtime.Format(TimeDisplay),
				})
				countNum++
				// TODO 格式化后写入文件作为报表
				//log.Println(key, duplicateMap[key])

			case n, _ := <-num:
				atomic.AddInt64(&atomicCount, n)
				//547432
				if count == atomicCount {
					done <- "done"
					return
				}
			default:

			}
		}
	}()

	fmt.Println(<-done)

	path := fmt.Sprintf("/opt/frs/%s/report.frs", root)
	generateReportContent(path, duplicateMap, "PERMISSION", 0)

	return countNum
}

// Suffix 按照后缀抽取数据
func Suffix(root string, count int64) int {
	countNum := 0
	maxConcurrency := SystemMaxConcurrency() // 最大并发打开文件数

	results := make(chan Duplicate)               // 创建一个通道来接收 重复数据 结果
	num := make(chan int64)                       // 创建一个通道来接收 重复数据 结果
	fileChan := make(chan Target, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

	// 启动多个 goroutine 处理文件
	for i := 0; i < maxConcurrency-1; i++ {
		go func() {
			for target := range fileChan {
				filter(target,
					// 自定义的 过滤条件
					func(target Target) (bool, string) {
						_, fileName := filepath.Split(target.Path)
						extension := false
						extensionName := ""

						for _, filter := range filters {
							matched := matchSuffix(fileName, filter)

							if matched {
								extension = true
								extensionName = filter
								break // 跳出循环
							}
						}

						return extension, extensionName
					},
					results, num)
			}
		}()
	}

	go func() {
		kvSourceDB := GlobalDatabaseKeyValueCache.Database
		kvBatch := kvSourceDB.NewIndexedBatch()
		// 遍历数据库
		iter, _ := kvBatch.NewIter(nil) //prefixIterOptions([]byte(""))
		for iter.First(); iter.Valid(); iter.Next() {
			// Only keys beginning with "prefix" will be visited.
			key := iter.Key()
			valueBytes, err := iter.ValueAndErr()

			atomic.AddInt64(&atomicCountAll, 1)
			if err != nil {
				//log.Errorf("fetch value bytes %v", err)
				continue
			}
			var metadata Metadata
			json.Unmarshal(valueBytes, &metadata)
			fileChan <- Target{Path: string(key), Metadata: metadata}
		}

		iter.Close()
		kvBatch.Commit(pebble.Sync)
		kvBatch.Close()
	}()

	duplicateMap := make(map[string][]Output)
	done := make(chan string)

	go func() {
		for {
			select {
			case duplicated, _ := <-results:
				//atomic.AddInt64(&resultCount, 1)
				//fmt.Println(fmd5, notFoundPathCount+fmd5.Count, lineCount)
				_, fileName := path.Split(duplicated.FilePath) // 解析路径

				key := duplicated.Tag + ":" + fileName // 组合成 key
				duplicateMap[key] = append(duplicateMap[key], Output{
					Name:  duplicated.FilePath,
					Mtime: duplicated.Mtime.Format(TimeDisplay),
				})
				countNum++
				// TODO 格式化后写入文件作为报表
				//log.Println(key, duplicateMap[key])

			case n, _ := <-num:
				atomic.AddInt64(&atomicCount, n)
				//547432
				if count == atomicCount {
					done <- "done"
					return
				}
			default:

			}
		}
	}()

	fmt.Println(<-done)

	path := fmt.Sprintf("/opt/frs/%s/report.frs", root)
	generateReportContent(path, duplicateMap, "SUFFIX", 0)

	return countNum
}

// Content 按照文件内容抽取数据
func Content(root string, count int64) int {
	countNum := 0
	maxConcurrency := SystemMaxConcurrency() // 最大并发打开文件数

	results := make(chan Duplicate)               // 创建一个通道来接收 重复数据 结果
	num := make(chan int64)                       // 创建一个通道来接收 重复数据 结果
	fileChan := make(chan Target, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

	// 启动多个 goroutine 处理文件
	for i := 0; i < maxConcurrency-1; i++ {
		go func() {
			for target := range fileChan {
				filter(target,
					// 自定义的 过滤条件
					func(target Target) (bool, string) {
						isProtectedByPassword, matchTag, _ := check(target.Path)
						return isProtectedByPassword || strings.Contains(matchTag, "unmatched"), fmt.Sprintf("%v & %s", isProtectedByPassword, matchTag)
					},
					results, num)
			}
		}()
	}

	go func() {
		kvSourceDB := GlobalDatabaseKeyValueCache.Database
		kvBatch := kvSourceDB.NewIndexedBatch()
		// 遍历数据库
		iter, _ := kvBatch.NewIter(nil) //prefixIterOptions([]byte(""))
		for iter.First(); iter.Valid(); iter.Next() {
			// Only keys beginning with "prefix" will be visited.
			key := iter.Key()
			valueBytes, err := iter.ValueAndErr()

			atomic.AddInt64(&atomicCountAll, 1)
			if err != nil {
				//log.Errorf("fetch value bytes %v", err)
				continue
			}
			var metadata Metadata
			json.Unmarshal(valueBytes, &metadata)
			fileChan <- Target{Path: string(key), Metadata: metadata}
		}

		iter.Close()
		kvBatch.Commit(pebble.Sync)
		kvBatch.Close()
	}()

	duplicateMap := make(map[string][]Output)
	done := make(chan string)

	go func() {
		for {
			select {
			case duplicated, _ := <-results:
				//atomic.AddInt64(&resultCount, 1)
				//fmt.Println(fmd5, notFoundPathCount+fmd5.Count, lineCount)
				_, fileName := path.Split(duplicated.FilePath) // 解析路径

				key := duplicated.Tag + ":" + fileName // 组合成 key
				duplicateMap[key] = append(duplicateMap[key], Output{
					Name:  duplicated.FilePath,
					Mtime: duplicated.Mtime.Format(TimeDisplay),
				})
				countNum++
				// TODO 格式化后写入文件作为报表
				//log.Println(key, duplicateMap[key])

			case n, _ := <-num:
				atomic.AddInt64(&atomicCount, n)
				//547432
				if count == atomicCount {
					done <- "done"
					return
				}
			default:

			}
		}
	}()

	fmt.Println(<-done)

	path := fmt.Sprintf("/opt/frs/%s/report.frs", root)
	generateReportContent(path, duplicateMap, "CONTENT", 0)

	return countNum
}
