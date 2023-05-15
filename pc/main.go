package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"kiwislice/localimagereader/core"

	"github.com/gin-gonic/gin"
)

var (
	dirPath string
	port    int
)

// HTML靜態樣板FS
//
//go:embed static/templates/*
var staticTemplatesFS embed.FS

// 設定要從 command line 讀取的參數
// 這邊所設定的會在 -h 或者輸入錯誤時出現提示哦！
func init() {
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

	// 載入嵌入的HTML模板
	templates, err := template.ParseFS(staticTemplatesFS, "static/templates/*.html")
	if err != nil {
		panic(err)
	}
	// 使用Gin的LoadHTMLGlob載入HTML模板
	r.SetHTMLTemplate(templates)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/fs/*subpath", func(c *gin.Context) {
		type btn struct {
			Label, Url string
		}
		subpath := c.Param("subpath")

		buttons := []btn{
			{
				Label: "回上一頁",
				Url:   fmt.Sprintf("/fs/%s", filepath.Dir(subpath)),
			},
		}
		imageUrls := []string{}

		for _, fi := range dirFs.GetDirContents(subpath) {
			if fi.IsDir {
				b := btn{
					Label: fi.Subpath,
					Url:   fmt.Sprintf("/fs/%s", fi.Subpath),
				}
				buttons = append(buttons, b)
			} else {
				imageUrls = append(imageUrls, fmt.Sprintf("/file/%s", fi.Subpath))
			}
		}

		c.HTML(http.StatusOK, "dir.html", gin.H{
			"buttons":   buttons,
			"imageUrls": imageUrls,
		})
	})

	r.Static("/file", dirFs.DirPath)

	homepageUrl := fmt.Sprintf("http://localhost:%d/fs", port)
	openbrowser(homepageUrl)

	r.Run(fmt.Sprintf(":%d", port)) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

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
