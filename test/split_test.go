package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
)

type WatchedStat struct {
	Ip         string `json:"ip"`
	Pid        string `json:"pid"`
	ServerPort string `json:"serverPort"`
	ClientPort string `json:"clientPort"`
	FilePath   string `json:"filePath"`
	AttackTime string `json:"attackTime"`
	Operation  string `json:"operation"`
	CmdLine    string `json:"cmdLine"`
}

func TestConv(t *testing.T) {
	tests := []struct {
		key    string
		source interface{}
		want   bool
	}{
		{`[{"clientPort":"49466","cmdLine":" tail -f /opt/baits/___aaa-password-rec","filePath":"/opt/baits/___aaa-password-rec","ip":"10.0.0.13","operation":"OPEN","pid":"1095732","serverPort":"22","attackTime":"2023-06-28 11:23:45"}]`, 10000000.0, true},
	}

	var columns []string
	var whereCondition []string

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {

			log.Println(fmt.Sprintf("watched state data : %s", tt.key))

			var watchedStatList []WatchedStat
			json.Unmarshal([]byte(tt.key), &watchedStatList)

			fmt.Println(watchedStatList)

			//columnNameValueArray := strings.SplitN(tt.key, ":", 2)
			//columnName := columnNameValueArray[0]
			//columnValue := columnNameValueArray[1]
			//
			//columns = append(columns, columnName)
			//if columnValue == "" {
			//	whereCondition = append(whereCondition, fmt.Sprintf(`( %s = '' or %s is null )`, columnName, columnName))
			//} else {
			//	whereCondition = append(whereCondition, strings.Replace(tt.key, ":", "='", 1)+"'")
			//}

		})
	}

	fmt.Println(strings.Join(columns, ","))
	fmt.Println(strings.Join(whereCondition, " and "))
}
