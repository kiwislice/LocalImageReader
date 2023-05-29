package core

import (
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

const thumbnailDirName = ".thumbnail" // 縮圖資料夾名稱
const thumbnailWidth int = 200        // 縮圖寬度

// 取得縮圖完整路徑
func (x *DirFileSystem) ThumbnailPath(subpath string) string {
	return filepath.Join(x.DirPath, thumbnailDirName, subpath)
}

// 取得資料夾中第一個predicate判斷為true的檔案，可能為nil
func (x *DirFileSystem) FindThumbnail(fullpath, subpath string) (thumbnailFullpath string) {
	// 先判斷縮圖是否存在，存在直接return
	thumbnailFullpath = x.ThumbnailPath(subpath)
	_, err := os.Stat(thumbnailFullpath)
	if err == nil {
		return thumbnailFullpath
	}

	ifErr := func(err error) {
		log.Fatal(err)
	}

	thumbnailFullpath = toJpeg(fullpath, thumbnailFullpath, ifErr)
	return thumbnailFullpath
}

// src=原圖完整路徑
// dest=縮圖完整路徑
// return=要顯示的檔案的完整路徑
func toJpeg(src, dest string, ifErr func(error)) string {
	img, err := readAsImage(src)
	if err != nil {
		err = fmt.Errorf("readAsImage失敗: %v", err)
		ifErr(err)
		return src
	}

	// 判斷寬度是否需要縮圖，不需要則return
	width := img.Bounds().Dx()
	if width <= thumbnailWidth {
		return src
	}

	writer, err := createThumbnailFile(dest)
	if err != nil {
		err = fmt.Errorf("createThumbnailFile失敗: %v", err)
		ifErr(err)
		return src
	}
	defer writer.Close()

	newImg := resizeImage(img, thumbnailWidth)
	err = jpeg.Encode(writer, newImg, nil)
	if err != nil {
		err = fmt.Errorf("jpeg.Encode失敗: %v", err)
		ifErr(err)
		return src
	}
	return dest
}

// 讀取Image物件
func readAsImage(fullpath string) (img image.Image, err error) {
	reader, err := os.Open(fullpath)
	if err != nil {
		err = fmt.Errorf("os.Open失敗: %v", err)
		return
	}
	defer reader.Close()

	img, _, err = image.Decode(reader)
	if err != nil {
		err = fmt.Errorf("image.Decode失敗: %v", err)
	}
	return
}

// 新增縮圖檔
func createThumbnailFile(thumbnailFullpath string) (*os.File, error) {
	dirPath := filepath.Dir(thumbnailFullpath)
	err := os.MkdirAll(dirPath, fs.ModeDir|fs.ModeTemporary)
	if err != nil {
		err = fmt.Errorf("os.MkdirAll失敗: %v", err)
		return nil, err
	}

	writer, err := os.Create(thumbnailFullpath)
	if err != nil {
		err = fmt.Errorf("os.Create失敗: %v", err)
		return nil, err
	}
	return writer, nil
}

// 圖片保持比例縮放到指定寬度
func resizeImage(img image.Image, newWidth int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 计算缩放比例
	hwRatio := float64(height) / float64(width)
	newHeight := int(hwRatio * float64(newWidth))

	// 创建缩小后的图像画布
	resizedImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// 使用双线性插值算法进行图像缩小
	draw.CatmullRom.Scale(resizedImg, resizedImg.Bounds(), img, bounds, draw.Over, nil)

	return resizedImg
}
