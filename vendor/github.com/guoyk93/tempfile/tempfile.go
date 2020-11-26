package tempfile

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	tempFiles []string
	tempDirs  []string

	mutex = &sync.Mutex{}
)

func allocateFilename(pfx, sfx string) (filename string, err error) {
	mutex.Lock()
	defer mutex.Unlock()
	var buf [6]byte
	if _, err = rand.Read(buf[:]); err != nil {
		return
	}
	filename = filepath.Join(os.TempDir(), pfx+"-"+time.Now().Format("20060102150405")+"-"+hex.EncodeToString(buf[:])+sfx)
	tempFiles = append(tempFiles, filename)
	return
}

func allocateDir(pfx string) (dirname string, err error) {
	mutex.Lock()
	defer mutex.Unlock()
	var buf [6]byte
	if _, err = rand.Read(buf[:]); err != nil {
		return
	}
	dirname = filepath.Join(os.TempDir(), pfx+"-"+time.Now().Format("20060102150405")+"-"+hex.EncodeToString(buf[:]))
	if err = os.MkdirAll(dirname, 0755); err != nil {
		return
	}
	tempDirs = append(tempDirs, dirname)
	return
}

func WriteDirFile(data []byte, dirPfx, name string, exe bool) (dirname, filename string, err error) {
	if dirname, err = allocateDir(dirPfx); err != nil {
		return
	}
	var mode os.FileMode
	if exe {
		mode = 0755
	} else {
		mode = 0644
	}
	filename = filepath.Join(dirname, name)
	err = ioutil.WriteFile(filename, data, mode)
	return
}

func WriteFile(data []byte, pfx, sfx string, exe bool) (filename string, err error) {
	if filename, err = allocateFilename(pfx, sfx); err != nil {
		return
	}
	var mode os.FileMode
	if exe {
		mode = 0755
	} else {
		mode = 0644
	}
	err = ioutil.WriteFile(filename, data, mode)
	return
}

func DeleteAll() {
	mutex.Lock()
	defer mutex.Unlock()
	for _, filename := range tempFiles {
		_ = os.Remove(filename)
	}
	tempFiles = []string{}
	for _, dirname := range tempDirs {
		_ = os.RemoveAll(dirname)
	}
	tempDirs = []string{}
	return
}
