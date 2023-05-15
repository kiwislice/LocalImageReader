package core

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type DirFileSystem struct {
	DirPath string
}

// 建立檔案系統
func NewDirFileSystem(path string) (fs *DirFileSystem, err error) {
	suc, err := checkPathDirExist(path)
	if suc {
		fs = &DirFileSystem{path}
	}
	return fs, err
}

// 取得所有檔案
func (x *DirFileSystem) Files() <-chan string {
	ch := make(chan string, 10)
	go func() {
		err := filepath.Walk(x.DirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(x.DirPath, path)
				if err != nil {
					return err
				}
				relPath = filepath.ToSlash(relPath)
				ch <- relPath
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
		close(ch)
	}()
	return ch
}

// 檢查路徑存在且是資料夾
func (x *DirFileSystem) Exists(subpath string) (suc bool, fi *FileInfo) {
	fullpath := filepath.Join(x.DirPath, subpath)
	fileInfo, err := os.Stat(fullpath)
	if err != nil {
		err = fmt.Errorf("無法取得路徑 %s 的檔案資訊： %v", fullpath, err)
		log.Println(err)
		return false, nil
	}
	return true, newFileInfo(fullpath, subpath, fileInfo.IsDir())
}

func (x *DirFileSystem) FullPath(subpath string) (fullpath string) {
	fullpath = filepath.Join(x.DirPath, subpath)
	return
}

func (x *DirFileSystem) GetDirContents(subpath string) []*FileInfo {
	exists, fi := x.Exists(subpath)
	if !exists {
		return []*FileInfo{}
	}
	if !fi.IsDir {
		return []*FileInfo{fi}
	}

	fs, _ := ioutil.ReadDir(fi.Fullpath)
	fis := make([]*FileInfo, 0, len(fs))
	for _, f := range fs {
		fullpath := filepath.Join(fi.Fullpath, f.Name())
		subpath := filepath.Join(subpath, f.Name())
		fi := newFileInfo(fullpath, subpath, f.IsDir())
		fis = append(fis, fi)
	}
	return fis
}

type FileInfo struct {
	Fullpath string
	Subpath  string
	IsDir    bool
}

func newFileInfo(fullpath, subpath string, isDir bool) *FileInfo {
	fi := new(FileInfo)
	fi.Fullpath = filepath.ToSlash(fullpath)
	fi.Subpath = filepath.ToSlash(subpath)
	fi.IsDir = isDir
	return fi
}

// 檢查路徑存在且是資料夾
func checkPathDirExist(path string) (suc bool, err error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		err = fmt.Errorf("無法取得路徑 %s 的檔案資訊： %v\n", path, err)
		return false, err
	}

	if fileInfo.IsDir() {
		fmt.Printf("%s 是一個資料夾\n", path)
		return true, err
	} else {
		err = fmt.Errorf("%s 不是一個資料夾\n", path)
		return false, err
	}
}
