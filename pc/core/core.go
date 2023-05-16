package core

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
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
		err := filepath.WalkDir(x.DirPath, func(path string, info os.DirEntry, err error) error {
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

// 相對路徑轉完整路徑
func (x *DirFileSystem) FullPath(subpath string) (fullpath string) {
	fullpath = filepath.Join(x.DirPath, subpath)
	return
}

// 取得資料夾內容，如果subpath不存在則空陣列，如果是檔案則只有檔案本身
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

// 取得資料夾中第一個predicate判斷為true的檔案，可能為nil
func (x *DirFileSystem) Find(subpath string, predicate func(fullpath string) bool) *FileInfo {
	exists, fi := x.Exists(subpath)
	if !exists {
		return nil
	}
	if !fi.IsDir {
		return fi
	}

	fs, _ := ioutil.ReadDir(fi.Fullpath)
	for _, f := range fs {
		fullpath := filepath.Join(fi.Fullpath, f.Name())
		subpath := filepath.Join(subpath, f.Name())
		if predicate(fullpath) {
			return newFileInfo(fullpath, subpath, f.IsDir())
		}
	}
	return nil
}

// 取得資料夾中第一個predicate判斷為true的檔案，可能為nil
func (x *DirFileSystem) FindRecursive(subpath string, predicate func(fullpath string) bool) *FileInfo {
	exists, fi := x.Exists(subpath)
	if !exists {
		return nil
	}
	if !fi.IsDir {
		return fi
	}

	waitQueue := make([]string, 0, 100) // 待處理資料夾
	waitQueue = append(waitQueue, fi.Fullpath)

	for len(waitQueue) > 0 {
		dirFullpath := waitQueue[0]
		waitQueue = waitQueue[1:]

		fs, _ := os.ReadDir(dirFullpath)
		for _, f := range fs {
			subFullpath := filepath.Join(dirFullpath, f.Name())
			if f.IsDir() {
				waitQueue = append(waitQueue, subFullpath)
				continue
			}
			if predicate(f.Name()) {
				relPath, err := filepath.Rel(x.DirPath, subFullpath)
				if err != nil {
					err = fmt.Errorf("取得相對路徑失敗：資料夾=%s, 檔案=%s, err=%v", x.DirPath, subFullpath, err)
					log.Println(err)
				}
				return newFileInfo(subFullpath, relPath, f.IsDir())
			}
		}
	}
	return nil
}

// 檔案資訊
type FileInfo struct {
	Fullpath string // 完整路徑
	Subpath  string //相對路徑
	IsDir    bool   //是否為資料夾
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

// 圖檔的附檔名
var imageExtentions = []string{".jpeg", ".jpg", ".png", ".gif", ".bmp", ".tiff", ".tif", ".webp", ".svg", ".ico"}

// 判斷路徑是否為圖檔
func IsImage(path string) bool {
	s := strings.ToLower(path)
	for i := 0; i < len(imageExtentions); i++ {
		if strings.HasSuffix(s, imageExtentions[i]) {
			return true
		}
	}
	return false
}
