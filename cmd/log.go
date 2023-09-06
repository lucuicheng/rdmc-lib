package main

import (
	"bufio"
	"fmt"
	"os"
)

type RedoLogRecord struct {
	SCN            int64
	ThreadID       int
	Sequence       int
	LogSequence    int
	BlockID        int
	RowID          string
	TableSpaceName string
	SegOwner       string
	SQLRedo        string
}

func main() {
	f, err := os.Open("tmp/redo01.log")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	var records []RedoLogRecord
	var currentRecord *RedoLogRecord

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if isRecordStart(line) {
			if currentRecord != nil {
				records = append(records, *currentRecord)
			}
			currentRecord = &RedoLogRecord{}
			parseRecordHeader(line, currentRecord)
		} else if isSQLRedo(line) {
			parseSQLRedo(line, currentRecord)
		}
	}

	if currentRecord != nil {
		records = append(records, *currentRecord)
	}

	for _, r := range records {
		fmt.Printf("SCN:%d, ThreadID:%d, Sequence:%d, LogSequence:%d, BlockID:%d, RowID:%s, TableSpaceName:%s, SegOwner:%s, SQLRedo:%s\n", r.SCN, r.ThreadID, r.Sequence, r.LogSequence, r.BlockID, r.RowID, r.TableSpaceName, r.SegOwner, r.SQLRedo)
	}
}

func isRecordStart(line string) bool {
	//return line[:3] == "###"
	return true
}

func isSQLRedo(line string) bool {
	//return line[:4] == "SQL> "
	return true
}

func parseRecordHeader(line string, record *RedoLogRecord) {
	//regex := regexp.MustCompile(`### SCN (\d+), Thread (\d+), Seq# (\d+), LogSeq# (\d+), Block (\d+), Slot (\d+), Row (\S+), Txn ID (\S+), SCN (\d+), Tablespace (\S+), SegOwner (\S+)`)
	//matches := regex.FindStringSubmatch(line)
	//
	//record.SCN, _ = strconv.ParseInt(matches[1], 10, 64)
	//record.ThreadID, _ = strconv.Atoi(matches[2])
	//record.Sequence, _ = strconv.Atoi(matches[3])
	//record.LogSequence, _ = strconv.Atoi(matches[4])
	//record.BlockID, _ = strconv.Atoi(matches[5])
	//record.RowID = matches[7]
	//record.TableSpaceName = matches[10]
	//record.SegOwner = matches[11]
	fmt.Println(line)
}

func parseSQLRedo(line string, record *RedoLogRecord) {
	//record.SQLRedo += line[5:] + " "
	fmt.Println(line)
}
