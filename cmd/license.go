package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hyperboloide/lk"
	"log"
	"os"
	"rdmc/pkg"
	"time"
)

type ActiveIOFRSLicence struct {
	IssueDate      time.Time `json:"issueDate"`
	Email          string    `json:"email"`
	User           string    `json:"user"`
	Company        string    `json:"company"`
	AgentCount     int       `json:"agentCount"`
	TaskCount      int       `json:"taskCount"`
	ValidityPeriod time.Time `json:"validityPeriod"`
	ExpirationDate time.Time `json:"expirationDate"`
}

func generate() (string, string) {
	// create a new Private key:
	privateKeyInfo, err := lk.NewPrivateKey()
	if err != nil {
		log.Fatal(err)
	}

	privateKey, _ := privateKeyInfo.ToB64String()
	fmt.Printf("privateKey: %s \n", privateKey)

	// get the public key. The public key should be hardcoded in your app to check licences.
	// Do not distribute the private key!
	publicKeyInfo := privateKeyInfo.GetPublicKey()
	publicKey := publicKeyInfo.ToB64String()
	fmt.Printf("publicKey %s \n", publicKeyInfo.ToB64String())

	return privateKey, publicKey

}

// 生成 可选 密钥公钥对，随机选择
const (
	PrivateKey = "KP+BAwEBC3BrQ29udGFpbmVyAf+CAAECAQNQdWIBCgABAUQB/4QAAAAK/4MFAQL/hgAAAP+Z/4IBYQSeME2llzJYV5gfrz6DpeXbbGyakysPZGdh73YDX9m3OmdukkRUYKEjgXXmGVdyh5S/cTmeHF4OG9LykMS0mPkzYPnjzSXNE4VM1wL1gtvnve6QnyV3EQKTo8NKpO8YLz0BMQIHFMP79akQIXHuLst8ebserCzDMX1yPbdo778s//rkq73pxXRVvWGymkzdqWdCqeUA"
	PublicKey  = "BJ4wTaWXMlhXmB+vPoOl5dtsbJqTKw9kZ2HvdgNf2bc6Z26SRFRgoSOBdeYZV3KHlL9xOZ4cXg4b0vKQxLSY+TNg+ePNJc0ThUzXAvWC2+e97pCfJXcRApOjw0qk7xgvPQ=="
)

func create(licenseInfo ActiveIOFRSLicence) string {
	// first, you need a base64 encoded private key generated by `lkgen gen` note that you might
	// prefer reading it from a file, and that it should stay secret (ie: dont distribute it with your app)!
	var privateKeyBase64 = PrivateKey

	// Unmarshal the private key
	privateKey, err := lk.PrivateKeyFromB64String(privateKeyBase64)
	if err != nil {
		log.Fatal(err)
	}

	// Define the data you need in your license,
	// here we use a struct that is marshalled to json, but ultimately all you need is a []byte.

	// marshall the document to []bytes (this is the data that our license will contain).
	docBytes, err := json.Marshal(licenseInfo)
	if err != nil {
		log.Fatal(err)
	}

	// generate your license with the private key and the document
	license, err := lk.NewLicense(privateKey, docBytes)
	if err != nil {
		log.Fatal(err)

	}
	// the b64 representation of our license, this is what you give to your customer.
	licenseB64, err := license.ToB64String()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(licenseB64)

	return licenseB64
}

func validate(licenseB64 string, expiration time.Time) {
	// A previously generated license b64 encoded. In real life you should read it from a file...

	// the public key b64 encoded from the private key using: lkgen pub my_private_key_file`.
	// It should be hardcoded somewhere in your app.
	var publicKeyBase64 = PublicKey

	// Unmarshal the public key.
	publicKey, err := lk.PublicKeyFromB64String(publicKeyBase64)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the customer license.
	license, err := lk.LicenseFromB64String(licenseB64)
	if err != nil {
		log.Fatal(err)
	}

	// validate the license signature.
	if ok, err := license.Verify(publicKey); err != nil {
		log.Fatal(err)
	} else if !ok {
		log.Fatal("Invalid license signature")
	}

	licenseInfo := ActiveIOFRSLicence{}

	// unmarshal the document.
	if err := json.Unmarshal(license.Data, &licenseInfo); err != nil {
		log.Fatal(err)
	}

	// Now you just have to check that the end date is after time.Now() then you can continue!

	if licenseInfo.ValidityPeriod.Before(expiration) { //
		log.Fatalf("License expired on: %s", licenseInfo.ValidityPeriod.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf(`Licensed to %v until %s`, licenseInfo, licenseInfo.ValidityPeriod.Format("2006-01-02 15:04:05"))
	}
}

func main() {

	dateString := flag.String("date", "2024-08-11 14:42:39", "Expiration Date")
	version := flag.Bool("v", false, "Prints current tool version")
	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - License For FRS v%s", pkg.AppVersion))
		os.Exit(0)
		return
	}

	current := time.Now()

	//dateString := "2023-08-10 14:42:39" // TODO 应该位系统内置安装时间

	// 将字符串解析为时间对象
	var cstZone = time.FixedZone("UTC", 8*3600) // 东八区
	expiration, err := time.ParseInLocation("2006-01-02 15:04:05", *dateString, cstZone)
	//expiration = expiration.In(cstZone)

	if err != nil {
		log.Fatalf("Error parsing date: %v\n", err)
		return
	}

	licenseInfo := ActiveIOFRSLicence{
		current,                           // 签发日期
		"test@example.com",                // 用户邮箱
		"admin@localhost",                 // 用户名称
		"ActiveIO",                        // 公司名称
		10,                                // 可安装 agent 数
		15,                                // 可执行总任务数
		current.Add(time.Hour * 24 * 365), // 有效期限, 实际应该是签发日前往后推算,但是签发日期 和 结束日期可以不匹配，
		expiration,                        // 过期时间（强制）， 用于控制 license 最后有效期限
	}

	licenseB64 := create(licenseInfo)

	validate(licenseB64, expiration)
	// TODO 解析后 比较 签发时间
}

/**
LP+BAwEBB0xpY2Vuc2UB/4IAAQMBBERhdGEBCgABAVIB/4QAAQFTAf+EAAAACv+DBQEC/4YAAAD+AWX/ggH/+XsiaXNzdWVEYXRlIjoiMjAyMy0wOC0yMlQxMDoyMjoxNi4zMzg0MDgrMDg6MDAiLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJ1c2VyIjoiYWRtaW5AbG9jYWxob3N0IiwiY29tcGFueSI6IkFjdGl2ZUlPIiwiYWdlbnRDb3VudCI6MTAsInRhc2tDb3VudCI6MTUsInZhbGlkaXR5UGVyaW9kIjoiMjAyNC0wOC0yMVQxMDoyMjoxNi4zMzg0MDgrMDg6MDAiLCJleHBpcmF0aW9uRGF0ZSI6IjIwMjQtMDgtMTFUMTQ6NDI6MzkrMDg6MDAifQExAm1PldhFmiBEt9m1M8t5tpGEsKQfuVFSDLRxKmJJnrVJhI1AB/mTX+CkE/gM77xRHgExAgh6ned1WnFbxNYaGikwcCq+44ZyDVn4cmW10xWfRK3SDzARNQFaG/OMUvrT4mi8fAA=
*/
