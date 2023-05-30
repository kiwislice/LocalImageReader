package core

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	cwebpExePath    string // cwebp.exe的完整路徑
	inputCwebpFlags string // cwebp.exe的參數
)

// 圖檔轉webp
//
//go:embed static/cwebp.exe
var cwebpFS embed.FS

// 設定要從 command line 讀取的參數
// 這邊所設定的會在 -h 或者輸入錯誤時出現提示哦！
func init() {
	flag.StringVar(&inputCwebpFlags, "cwebpflags", "", "執行cwebp.exe的參數")
}

// 取得webp完整路徑
func (x *DirFileSystem) WebpPath(subpath string) string {
	ext := filepath.Ext(subpath)
	webpPath := subpath[:len(subpath)-len(ext)] + ".webp"
	return filepath.Join(x.DirPath, thumbnailDirName, webpPath)
}

// 取得webp的檔案，可能為nil
func (x *DirFileSystem) FindWebp(fullpath, subpath string) (webpFullpath string) {
	if !IsImage(subpath) {
		return fullpath
	}

	// 先判斷webp是否存在，存在直接return
	webpFullpath = x.WebpPath(subpath)
	_, err := os.Stat(webpFullpath)
	if err == nil {
		return webpFullpath
	}

	ifErr := func(err error) {
		log.Fatal(err)
	}

	webpFullpath = toWebp(fullpath, webpFullpath, ifErr)
	return webpFullpath
}

const cwebpexeName = "cwebp.exe"

func init() {
	// 獲取當前執行檔案的絕對路徑
	exePath, _ := os.Executable()
	fmt.Println("當前執行檔案的路徑：", exePath)

	dirPath := filepath.Dir(exePath)
	// 構建可執行文件的絕對路徑
	cwebpExePath = filepath.Join(dirPath, cwebpexeName)
	fmt.Println("cwebpPath：", cwebpExePath)

	createCwebpexe(cwebpExePath)
}

func createCwebpexe(fullpath string) {
	// 將exe檔案寫入文件
	exeFile, err := os.Create(fullpath)
	if err != nil {
		fmt.Println("無法創建臨時文件：", err)
	}
	defer exeFile.Close()

	exeData, err := cwebpFS.ReadFile("static/" + cwebpexeName)
	if err != nil {
		fmt.Println("無法讀取嵌入的exe檔案：", err)
	}

	if _, err := exeFile.Write(exeData); err != nil {
		fmt.Println("無法寫入臨時exe文件：", err)
	}
}

// src=原圖完整路徑
// dest=webp完整路徑
// return=要顯示的檔案的完整路徑
func toWebp(src, dest string, ifErr func(error)) string {
	_, err := os.Stat(dest)
	if err == nil {
		return dest
	}

	dirPath := filepath.Dir(dest)
	os.MkdirAll(dirPath, fs.ModeDir)

	_, err = os.Stat(cwebpExePath)
	if err != nil {
		fmt.Printf("toWebp cwebpexe不存在：%v\n", err)
		createCwebpexe(cwebpExePath)
	}

	cwebpFlags := []string{"-q", "50", src, "-o", dest}
	inputCwebpFlags = strings.TrimSpace(inputCwebpFlags)
	if inputCwebpFlags != "" {
		cwebpFlags = strings.Split(inputCwebpFlags, " ")
		cwebpFlags = append(cwebpFlags, src, "-o", dest)
	}

	// 執行臨時exe檔案
	cmd := exec.Command(cwebpExePath, cwebpFlags...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("toWebp執行exe檔案時出錯：", err, "flags=", cwebpFlags)
		return src
	}
	return dest
}
