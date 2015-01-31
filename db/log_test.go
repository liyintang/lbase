package db

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// create a slice that is long enough to across log file boundary
func MakeReallyLongRecord(howLong int) []byte {
	ret := make([]byte, howLong)
	var ch byte
	for idx,_ := range ret {
		ret[idx] = ch
		ch++
	}

	return ret
}

func TestLoggerWriteAndReadBack(t *testing.T) {
	root := "/tmp/logger_test/LoggerWriteAndReadBack"

	os.RemoveAll(root)
	os.MkdirAll(root, os.ModePerm)

	// create a new file and writer
	name := strings.Join([]string{root, "log"}, "/")
	wf := MakeLocalWritableFile(name)

	if wf == nil {
		t.Error("Fails to create a new log file ", name)
	}

	writer := LogWriter{wf}

	// prepare records to be appended
	strRecords := []string{"hello, world", "go programming", "key value"}
	records := [][]byte{}

	for _, s := range strRecords {
		records = append(records, []byte(s))
	}

	// append records
	for _, r := range records {
		ok := writer.AddRecord(r)
		if !ok.Ok() {
			t.Error("Fails to append a record!")
		}
	}

	wf.Close()

	// open the file for read
	rf := MakeLocalSequentialFile(name)
	if rf == nil {
		t.Error("Fail to open a file for read")
	}

	reader := LogReader{rf, int64(0), true}
	buf := make([]byte, 2048)

	// read and validate the records
	for i, r := range records {
		ret, status := reader.ReadRecord(buf)
		if status != ReadStatusOk {
			t.Error("Fails to read from a log file")
		}

		if bytes.Compare(ret, r) != 0 {
			t.Error("Fails to read the exactly same record", i)
		}
	}

	_, status := reader.ReadRecord(buf)
	if status != ReadStatusEOF {
		t.Error("Suppose to end at this point")
	}
}

func TestReaderWriterAcrossSingleBlock(t *testing.T) {
	root := "/tmp/logger_test/ReaderWriterAcrossSingleBlock"

	os.RemoveAll(root)
	os.MkdirAll(root, os.ModePerm)

	// create a new file and writer
	name := strings.Join([]string{root, "log"}, "/")
	wf := MakeLocalWritableFile(name)

	if wf == nil {
		t.Error("Fails to create a new log file ", name)
	}

	writer := LogWriter{wf}

	// prepare records to be appended
	firstRecord := "hello world"
	secondRecord := MakeReallyLongRecord(kBlockSize+2)
	thirdRecord := "go programming is fun"

	records := [][]byte{
		[]byte(firstRecord),
		secondRecord,
		[]byte(thirdRecord),
	}

	// append records
	for _, r := range records {
		ok := writer.AddRecord(r)
		if !ok.Ok() {
			t.Error("Fails to append a record!")
		}
	}

	wf.Close()

	// open the file for read
	rf := MakeLocalSequentialFile(name)
	if rf == nil {
		t.Error("Fail to open a file for read")
	}

	reader := LogReader{rf, int64(0), true}
	buf := make([]byte, 2*kBlockSize)

	// read and validate the records
	for i, r := range records {
		ret, status := reader.ReadRecord(buf)
		if status != ReadStatusOk {
			t.Error("Fails to read from a log file")
		}

		if bytes.Compare(ret, r) != 0 {
			t.Error("Fails to read the exactly same record", i)
		}
	}

	_, status := reader.ReadRecord(buf)
	if status != ReadStatusEOF {
		t.Error("Suppose to end at this point")
	}
}

func TestReaderWriterAcrossMultiBlock(t *testing.T) {
	root := "/tmp/logger_test/ReaderWriterAcrossMultiBlock"

	os.RemoveAll(root)
	os.MkdirAll(root, os.ModePerm)

	// create a new file and writer
	name := strings.Join([]string{root, "log"}, "/")
	wf := MakeLocalWritableFile(name)

	if wf == nil {
		t.Error("Fails to create a new log file ", name)
	}

	writer := LogWriter{wf}

	// prepare records to be appended
	firstRecord := "hello world"
	secondRecord := MakeReallyLongRecord(2*kBlockSize+2)
	thirdRecord := "go programming is fun"

	records := [][]byte{
		[]byte(firstRecord),
		secondRecord,
		[]byte(thirdRecord),
	}

	// append records
	for _, r := range records {
		ok := writer.AddRecord(r)
		if !ok.Ok() {
			t.Error("Fails to append a record!")
		}
	}

	wf.Close()

	// open the file for read
	rf := MakeLocalSequentialFile(name)
	if rf == nil {
		t.Error("Fail to open a file for read")
	}

	reader := LogReader{rf, int64(0), true}
	buf := make([]byte, 4*kBlockSize)

	// read and validate the records
	for i, r := range records {
		ret, status := reader.ReadRecord(buf)
		if status != ReadStatusOk {
			t.Error("Fails to read from a log file")
		}

		if bytes.Compare(ret, r) != 0 {
			t.Error("Fails to read the exactly same record", i)
		}
	}

	_, status := reader.ReadRecord(buf)
	if status != ReadStatusEOF {
		t.Error("Suppose to end at this point")
	}
}
