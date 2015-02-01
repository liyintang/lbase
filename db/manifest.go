package db

import (
	"bytes"
	"encoding/gob"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

const (
	ManifestPrefix = "manifest_"
)

// In-memory representation of each nsst file.
// File may be deleted if refcnt reaches 0 (i.e. no iterator
// refers to it, and there is no explicit snapshot request
// that needs it)
type FileInfo struct {
	Location string
	BeginKey []byte
	EndKey   []byte
	Refcnt   int
}

// Each time a new file is created or an old file is deleted,
// the system creates a new snapshot. Old snapshot will be
// deleted if its refcnt becomes 0 (i.e. no iterator refers
// to it, and there is no explicit snapshot request for it)
type FileSnapshotInfo struct {
	Levels [][]int64
	Refcnt int
}

// In memory representation of a manifest file. Each manifest
// file consist of an initial snapshot and logs of subsequent
// modifying request. On startup, old manifest file is read,
// logs in the file are replayed, and the resulting Manifest
// data structure is serialized to a new file as the base
// for next manifest file.
type ManifestData struct {
	FileMap          map[int64]FileInfo
	NextId           uint64
	FileSnapshotMap  map[int64]FileSnapshotInfo
	NextFileSnapshot int64
}

type Manifest struct {
	ManifestData
	env   Env
	rwMux sync.RWMutex
}

// Parse base file name, return its manifest number. If the base
// name does not fit into manifest file pattern, return -1 instead
func ParseManifestName(fname string) int64 {
	numPart := strings.TrimPrefix(fname, ManifestPrefix)
	if len(numPart) == len(fname) {
		return -1
	}

	numVal, err := strconv.ParseInt(numPart, 10, 64)
	if err != nil {
		return -1
	} else {
		return numVal
	}
}

// Helper type to sort slice of int64
type int64Sortee []int64

func (x int64Sortee) Len() int           { return len(x) }
func (x int64Sortee) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x int64Sortee) Less(i, j int) bool { return x[i] < x[j] }

// Return all manifest files in given directory @path. Then return
// full pathes of those files in ascending time order.
func ListAllManifestFiles(e Env, parent string) []string {
	lists, status := e.GetChildren(parent)
	if !status.Ok() {
		return []string{}
	}

	fileMap := make(map[int64]string)
	numList := make([]int64, 0, len(lists))

	for _, name := range lists {
		numVal := ParseManifestName(name)
		if numVal >= 0 {
			fileMap[numVal] = name
			numList = append(numList, numVal)
		}
	}

	sort.Sort(int64Sortee(numList))

	ret := make([]string, 0, len(numList))
	for _, num := range numList {
		val, ok := fileMap[num]
		if ok == true {
			ret = append(ret, path.Join(parent, val))
		}
	}

	return ret
}

func recoverSingleManifest(e Env, fullPath string) *Manifest {
	// first try to open the file
	ret := Manifest{env: e}
	file, status := e.NewSequentialFile(fullPath)
	if !status.Ok() {
		return nil
	}

	// read snapshot size from the file
	sizeBuf := make([]byte, 4)
	var dataReads []byte
	dataReads, status = file.Read(sizeBuf)
	if !status.Ok() {
		return nil
	}

	// read snapshot into buffer
	snapshotSize := *(*int32)(unsafe.Pointer(&dataReads[0]))
	dataSnapshot := make([]byte, snapshotSize)

	// use gob to decode it
	buffer := bytes.NewBuffer(dataSnapshot)
	dec := gob.NewDecoder(buffer)
	err := dec.Decode(&ret)
	if err != nil {
		return nil
	}

	return &ret
}

func initNewManifest(e Env, parent string) *Manifest {
	ret := Manifest{
		ManifestData: ManifestData{
			FileMap:         make(map[int64]FileInfo),
			FileSnapshotMap: make(map[int64]FileSnapshotInfo),
		},
		env: e,
	}

	return &ret
}

func RecoverManifest(e Env, parent string, createIfMissing bool) *Manifest {
	paths := ListAllManifestFiles(e, parent)
	var ret *Manifest
	for i := len(paths) - 1; i >= 0; i-- {
		fullPath := paths[i]
		if ret == nil {
			tmp := recoverSingleManifest(e, fullPath)
			if tmp != nil {
				ret = tmp
				continue
			}
		}

		// remove corrupted or old manifest files
		e.DeleteFile(fullPath)
	}

	if ret == nil && createIfMissing {
		ret = initNewManifest(e, parent)
	}

	return ret
}
