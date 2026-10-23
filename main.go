package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const version = "1.0.0"

var serviceStop = make(chan struct{})

type config struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	DataDir      string `yaml:"data_dir"`
	OpenBrowser  bool   `yaml:"open_browser"`
	SecureCookie bool   `yaml:"secure_cookie"`
	LogFile      bool   `yaml:"log_file"`
}
type rotatingLog struct {
	mu   sync.Mutex
	path string
	file *os.File
}

func (l *rotatingLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, e := l.file.Stat()
	if e != nil {
		return 0, e
	}
	if s.Size()+int64(len(p)) > 5<<20 {
		if e = l.file.Close(); e != nil {
			return 0, e
		}
		os.Remove(l.path + ".3")
		for i := 2; i >= 1; i-- {
			os.Rename(fmt.Sprintf("%s.%d", l.path, i), fmt.Sprintf("%s.%d", l.path, i+1))
		}
		if e = os.Rename(l.path, l.path+".1"); e != nil {
			return 0, e
		}
		l.file, e = os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			return 0, e
		}
	}
	return l.file.Write(p)
}
func run() (runErr error) {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	port := flag.Int("port", 0, "覆盖 HTTP 端口")
	showVersion := flag.Bool("version", false, "显示版本")
	noBrowser := flag.Bool("no-browser", false, "关闭自动打开浏览器")
	flag.Parse()
	if *showVersion {
		fmt.Println("NavHub " + version)
		return nil
	}
	c := config{Host: "0.0.0.0", Port: 3002, DataDir: "data", OpenBrowser: true, LogFile: true}
	path, e := filepath.Abs(*configPath)
	if e != nil {
		return e
	}
	raw, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		raw, e = yaml.Marshal(c)
		if e != nil {
			return e
		}
		if e = os.WriteFile(path, raw, 0600); e != nil {
			return fmt.Errorf("创建配置: %w", e)
		}
	} else if e != nil {
		return e
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if e = decoder.Decode(&c); e != nil {
		return e
	}
	if *port != 0 {
		c.Port = *port
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("端口需在 1–65535 之间")
	}
	if c.DataDir == "" {
		return fmt.Errorf("data_dir 不能为空")
	}
	if !filepath.IsAbs(c.DataDir) {
		c.DataDir = filepath.Join(filepath.Dir(path), c.DataDir)
	}
	c.DataDir, e = filepath.Abs(c.DataDir)
	if e != nil {
		return e
	}
	for _, d := range []string{c.DataDir, filepath.Join(c.DataDir, "uploads"), filepath.Join(c.DataDir, "backups")} {
		if e = os.MkdirAll(d, 0700); e != nil {
			return e
		}
	}
	if c.LogFile {
		p := filepath.Join(c.DataDir, "navhub.log")
		f, e := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		writer := &rotatingLog{path: p, file: f}
		defer func() {
			if runErr != nil {
				log.Printf("启动 / 运行失败: %v", runErr)
			}
			log.SetOutput(os.Stdout)
			writer.file.Close()
		}()
		log.SetOutput(io.MultiWriter(writer, os.Stdout))
	} else {
		log.SetOutput(os.Stdout)
	}
	db, e := openDB(filepath.Join(c.DataDir, "navhub.db"))
	if e != nil {
		return fmt.Errorf("数据库迁移 / 初始化失败: %w", e)
	}
	defer db.Close()
	log.Print("数据库初始化完成，schema=1")
	listener, e := net.Listen("tcp", net.JoinHostPort(c.Host, fmt.Sprint(c.Port)))
	if e != nil {
		return fmt.Errorf("端口不可用: %w", e)
	}
	a := &app{db: db, dir: c.DataDir, secure: c.SecureCookie, attempts: map[string]attempt{}}
	server := &http.Server{Handler: a.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 2 * time.Minute, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	address := fmt.Sprintf("http://127.0.0.1:%d", c.Port)
	log.Printf("NavHub v%s | 运行中\n服务地址: %s\n监听地址: %s\n数据目录: %s", version, address, listener.Addr(), c.DataDir)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	if c.OpenBrowser && !*noBrowser {
		openBrowser(address)
	}
	select {
	case e = <-done:
		if e != http.ErrServerClosed {
			return e
		}
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if e = server.Shutdown(ctx); e != nil {
			return e
		}
	case <-serviceStop:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if e = server.Shutdown(ctx); e != nil {
			return e
		}
	}
	log.Print("服务已退出")
	return nil
}
func main() {
	if runAsService() {
		return
	}
	if e := run(); e != nil {
		log.Printf("启动 / 运行失败: %v", e)
		os.Exit(1)
	}
}
