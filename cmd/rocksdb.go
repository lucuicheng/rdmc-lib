package main

import (
	"fmt"
	"github.com/tecbot/gorocksdb"
)

func main() {
	// 创建 RocksDB 数据库选项
	options := gorocksdb.NewDefaultOptions()
	defer options.Destroy()

	// 设置 RocksDB 数据库选项
	options.SetCreateIfMissing(true)

	// 打开 RocksDB 数据库
	db, err := gorocksdb.OpenDb(options, "/tmp/rocksdb")
	if err != nil {
		fmt.Println("open db error: ", err)
		return
	}
	defer db.Close()

	// 写入数据
	writeOptions := gorocksdb.NewDefaultWriteOptions()
	defer writeOptions.Destroy()
	key := []byte("key")
	value := []byte("value")
	err = db.Put(writeOptions, key, value)
	if err != nil {
		fmt.Println("put error: ", err)
		return
	}

	// 读取数据
	readOptions := gorocksdb.NewDefaultReadOptions()
	defer readOptions.Destroy()
	data, err := db.Get(readOptions, key)
	if err != nil {
		fmt.Println("get error: ", err)
		return
	}
	defer data.Free()
	fmt.Println("value:", string(data.Data()))

	// 删除数据
	deleteOptions := gorocksdb.NewDefaultWriteOptions()
	defer deleteOptions.Destroy()
	err = db.Delete(deleteOptions, key)
	if err != nil {
		fmt.Println("delete error: ", err)
		return
	}
}
