package main

import (
	"crypto/rand"
	"embed"
	"flag"
	"fmt"
	"image"
	"io/fs"
	"math/big"
	"path"
	"sort"
	"strconv"
	"strings"

	"html/template"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"kiwislice/localimagereader/core"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"
	"golang.org/x/image/draw"
)

var (
	dirPath string
	port    int
	randStr string = randomString()
)

// HTML靜態樣板FS
//
//go:embed static/templates/*
var staticTemplatesFS embed.FS

// 設定要從 command line 讀取的參數
// 這邊所設定的會在 -h 或者輸入錯誤時出現提示哦！
func init() {
	fmt.Println(os.Args[0])
	curDir, _ := filepath.Split(os.Args[0])

	flag.StringVar(&dirPath, "dir", curDir, "資料夾路徑")
	flag.IntVar(&port, "port", 61091, "http port")

	flag.Usage = usage
}

func usage() {
	doc := `
本地檔案資料夾轉WEB瀏覽

localimagereader.exe [<args>]
	`
	fmt.Println(doc)
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	dirFs, err := core.NewDirFileSystem(dirPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("開放資料夾: %v\n", dirPath)

	r := gin.Default()

	LoadHtmlTemplateEmbed(r)
	// LoadHtmlTemplateGlobal(r)

	// 添加中間件以開放同源政策的限制
	r.Use(CORS())

	// 添加中間件以CACHE檔案下載
	r.Use(CACHE())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/fs/*subpath", func(c *gin.Context) {
		subpath := c.Param("subpath")
		wds := getWebData(dirFs, subpath)

		buttons := []webData{}
		imageUrls := []string{}

		for _, wd := range wds {
			if wd.IsDir {
				buttons = append(buttons, wd)
			} else {
				imageUrls = append(imageUrls, fmt.Sprintf("/file/%s", wd.Subpath))
			}
		}

		c.HTML(http.StatusOK, "dir.html", gin.H{
			"buttons":   buttons,
			"imageUrls": imageUrls,
		})
	})

	r.Static("/file", dirFs.DirPath)

	r.GET("/thumbnail/*subpath", func(c *gin.Context) {
		subpath := c.Param("subpath")
		fullpath := dirFs.FullPath(subpath)
		ginRouteThumbnailHandler(c, fullpath)
	})

	r.GET("/flutter/*subpath", func(c *gin.Context) {
		subpath := c.Param("subpath")
		wds := getWebData(dirFs, subpath)
		c.JSON(http.StatusOK, wds)
	})

	homepageUrl := fmt.Sprintf("http://localhost:%d/fs", port)
	openbrowser(homepageUrl)

	r.Run(fmt.Sprintf(":%d", port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

// 網頁用的JSON資料
type webData struct {
	IsDir    bool   // 是否為資料夾
	ImageUrl string // 如果是檔案則為本體，資料夾則為示意圖
	Label    string // 標題
	Subpath  string // 檔案系統的相對路徑
	FileName string // 完整檔名
}

func getWebData(dirFs *core.DirFileSystem, subpath string) []webData {
	array := make([]webData, 0, 100)
	// array = append(array, webData{
	// 	IsDir:    true,
	// 	ImageUrl: "",
	// 	Label:    "回上一頁",
	// 	Subpath:  filepathAdjust(filepath.Dir(subpath)),
	// })

	for _, fi := range dirFs.GetDirContents(subpath) {
		subpath := filepathAdjust(fi.Subpath)
		filaname := fi.FileName

		if fi.IsDir {
			b := webData{
				IsDir:    true,
				ImageUrl: "",
				Label:    subpath,
				Subpath:  subpath,
				FileName: filaname,
			}

			img := dirFs.FindRecursive(subpath, core.IsImage)
			if img != nil {
				b.ImageUrl = filepathAdjust(img.Subpath) + "?" + randStr
			}
			array = append(array, b)
		} else {
			array = append(array, webData{
				IsDir:    false,
				ImageUrl: filepathAdjust(fmt.Sprintf("/file/%s", subpath)) + "?" + randStr,
				Label:    subpath,
				Subpath:  subpath,
				FileName: filaname,
			})
		}
	}

	sort.SliceStable(array, func(i, j int) bool {
		return aLessBNumberFirst(array[i], array[j])
	})

	return array
}

// 純數字檔名優先
func aLessBNumberFirst(a, b webData) bool {
	// 获取扩展名（包括点号）
	aExt, bExt := filepath.Ext(a.FileName), filepath.Ext(b.FileName)
	aName, bName := strings.TrimSuffix(a.FileName, aExt), strings.TrimSuffix(b.FileName, bExt)
	aNum, aErr := strconv.Atoi(aName)
	bNum, bErr := strconv.Atoi(bName)

	if aErr != nil && bErr != nil {
		return strings.Compare(aName, bName) < 0
	}
	if ae, be := aErr != nil, bErr != nil; ae != be {
		if ae {
			return false
		} else {
			return true
		}
	}
	return aNum < bNum
}

// 調整路徑中的斜線&點
func filepathAdjust(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

// 開啟瀏覽器到指定url
func openbrowser(url string) {
	fmt.Println(url)
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

// 自訂的CORS中間件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// 自訂CACHE中間件
func CACHE() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("RequestURI=", c.Request.RequestURI)
		u, err := url.Parse(c.Request.RequestURI)
		if err != nil {
			fmt.Println("URL 解析失败:", err)
			return
		}
		fn := path.Base(u.Path)
		if core.IsImage(fn) {
			fmt.Println("Cache=", c.Request.RequestURI)
			c.Writer.Header().Set("Cache-Control", "public, max-age=600")
		}
		c.Next()
	}
}

// 使用嵌入static資料夾的模板
func LoadHtmlTemplateEmbed(r *gin.Engine) {
	// 載入嵌入的HTML模板
	templates, err := template.ParseFS(staticTemplatesFS, "static/templates/*.html")
	if err != nil {
		panic(err)
	}
	r.SetHTMLTemplate(templates)
}

// 使用隔壁static資料夾的模板
func LoadHtmlTemplateGlobal(r *gin.Engine) {
	// 使用Gin的LoadHTMLGlob載入HTML模板
	r.LoadHTMLGlob("static/templates/*.html")
}

func randomString() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	s := n.String()
	return s
}

// gin的/thumbnail處理
func ginRouteThumbnailHandler(c *gin.Context, fullpath string) {
	ifErr := func(err error) {
		log.Fatal(err)
		c.File(fullpath)
	}

	reader, err := os.Open(fullpath)
	if err != nil {
		err = fmt.Errorf("os.Open失敗: %v", err)
		ifErr(err)
		return
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		err = fmt.Errorf("image.Decode失敗: %v", err)
		ifErr(err)
		return
	}

	const thumbnailWidth int = 200

	width := img.Bounds().Dx()
	if width > thumbnailWidth {
		newImg := resizeImage(img, thumbnailWidth)

		// 不使用暫存檔
		err = jpeg.Encode(c.Writer, newImg, nil)
		if err != nil {
			err = fmt.Errorf("jpeg.Encode失敗: %v", err)
			ifErr(err)
			return
		}
		c.Writer.Header().Set("Content-Type", "image/jpeg")

		// 使用暫存檔
		// tempFile, err := newThumbnailFile()
		// if err != nil {
		// 	err = fmt.Errorf("newThumbnailFile失敗: %v", err)
		// 	ifErr(err)
		// 	return
		// }
		// err = jpeg.Encode(tempFile.Writer, newImg, nil)
		// if err != nil {
		// 	err = fmt.Errorf("jpeg.Encode失敗: %v", err)
		// 	ifErr(err)
		// 	return
		// }
		// tempFile.Writer.Close()
		// c.File(tempFile.FullPath)
	} else {
		c.File(fullpath)
	}
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

// 縮圖檔案
type thumbnailFile struct {
	Writer   *os.File
	FileName string
	FullPath string
}

// 新增暫存檔
func newThumbnailFile() (f *thumbnailFile, err error) {
	const dirPath = ".thumbnail"
	err = os.MkdirAll(dirPath, fs.ModeDir|fs.ModeTemporary)
	if err != nil {
		err = fmt.Errorf("os.MkdirAll失敗: %v", err)
		return
	}
	f = new(thumbnailFile)
	f.Writer, err = os.CreateTemp(dirPath, "thumbnail_*.jpg")
	if err != nil {
		err = fmt.Errorf("os.CreateTemp失敗: %v", err)
		return nil, err
	}

	info, err := f.Writer.Stat()
	if err != nil {
		err = fmt.Errorf("os.File.Stat失敗: %v", err)
		return nil, err
	}
	f.FileName = info.Name()
	f.FullPath = filepath.Join(dirPath, f.FileName)
	return
}
