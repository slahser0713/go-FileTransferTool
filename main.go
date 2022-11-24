package main

import (
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zserge/lorca"
)

//go:embed frontend/dist/*
var FS embed.FS

type Json struct {
	// json 字段重命名
	Raw string `json:"raw"`
}

func main() {
	var TextsController func(*gin.Context)
	// 获取文本并返回文件路径
	TextsController = func(c *gin.Context) {
		var json Json
		// 获取网页端输入文本(post请求的body)
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			exe, err := os.Executable() // 获取当前可执行文件的路径
			if err != nil {
				log.Fatal(err)
			}
			dir := filepath.Dir(exe) // 获取当前执行文件的目录
			if err != nil {
				log.Fatal(err)
			}
			filename := uuid.New().String()          // 生成一个文件名
			uploads := filepath.Join(dir, "uploads") // 拼接 uploads 的绝对路径
			err = os.MkdirAll(uploads, os.ModePerm)  // 创建 uploads 目录
			if err != nil {
				log.Fatal(err)
			}
			fullpath := path.Join("uploads", filename+".txt")                            // 拼接文件的绝对路径（不含 exe 所在目录）
			err = ioutil.WriteFile(filepath.Join(dir, fullpath), []byte(json.Raw), 0644) // 将 json.Raw 写入文件
			if err != nil {
				log.Fatal(err)
			}
			c.JSON(http.StatusOK, gin.H{"url": "/" + fullpath}) // 返回文件的绝对路径（不含 exe 所在目录）
		}
	}

	go func() { // gin 协程
		gin.SetMode(gin.DebugMode)
		router := gin.Default()
		staticFiles, _ := fs.Sub(FS, "frontend/dist")
		router.POST("/api/v1/texts", TextsController)
		router.StaticFS("/static", http.FS(staticFiles))
		router.NoRoute(func(c *gin.Context) {
			// 访问路径
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/static/") {
				reader, err := staticFiles.Open("index.html")
				if err != nil {
					log.Fatal(err)
				}
				defer reader.Close()
				stat, err := reader.Stat()
				if err != nil {
					log.Fatal(err)
				}
				c.DataFromReader(http.StatusOK, stat.Size(), "text/html;charset=utf-8", reader, nil)
			} else {
				c.Status(http.StatusNotFound)
			}
		})
		// 服务器连接至8080端口，服务器开始监听8080
		router.Run(":8080")
	}()
	ui, _ := lorca.New("http://127.0.0.1:8080/static/index.html", "", 1000, 800, "--disable-sync", "--disable-translate")
	chSignal := make(chan os.Signal, 1)
	signal.Notify(chSignal, syscall.SIGTERM, os.Interrupt)
	select {
	case <-ui.Done():
		fmt.Println("uiExit")
		// 监听到Ctrl+c
	case <-chSignal:
		fmt.Println("Ctrl+c")
	}
	ui.Close()
}
