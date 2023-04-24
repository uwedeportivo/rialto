package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/manucorporat/stats"
)

var ips = stats.New()

func rateLimit(c *gin.Context) {
	ip := c.ClientIP()
	value := int(ips.Add(ip, 1))
	if value%50 == 0 {
		fmt.Printf("ip: %s, count: %d\n", ip, value)
	}
	if value >= 200 {
		if value%200 == 0 {
			fmt.Println("ip blocked")
		}
		c.Abort()
		c.String(http.StatusServiceUnavailable, "you were automatically banned :)")
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)

	router := gin.New()
	router.Use(rateLimit, gin.Recovery())

	router.GET("/bridge/:gpath/*filename", func(c *gin.Context) {
		gpath := c.Param("gpath")
		filename := c.Param("filename")

		backendUrl := "https://drive.google.com/uc?export=view&id=" + gpath

		response, err := http.Get(backendUrl)
		if err != nil || response.StatusCode != http.StatusOK {
			c.Status(http.StatusServiceUnavailable)
			return
		}

		reader := response.Body
		defer func(reader io.ReadCloser) {
			err := reader.Close()
			if err != nil {
				log.Println("failed to close response body")
			}
		}(reader)
		contentLength := response.ContentLength
		contentType := response.Header.Get("Content-Type")

		extraHeaders := map[string]string{
			"Content-Disposition": fmt.Sprintf(`inline; filename="%s"`, filename),
		}

		c.DataFromReader(http.StatusOK, contentLength, contentType, reader, extraHeaders)
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil {
		log.Panicf("error: %s", err)
	}
}
