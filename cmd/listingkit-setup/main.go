// listingkit-setup is an explicit host tool, never part of business serving.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	runtimeSetup "task-processor/internal/app/runtime/setup"
	"task-processor/internal/app/setupcontrol"
	"task-processor/internal/platform/privatefile"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "安装管理未启动：请检查本机版本包、Docker和私密目录权限（未输出配置或凭据）")
		os.Exit(1)
	}
}
func run() error {
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	bundle := flag.String("package-dir", filepath.Dir(exe), "prepared local setup package (not a state-directory override)")
	flag.Parse()
	root, e := filepath.Abs(*bundle)
	if e != nil {
		return e
	}
	if e = privatefile.RejectPath(root); e != nil {
		return e
	}
	b, e := os.ReadFile(filepath.Join(root, "release.json"))
	if e != nil || len(b) > 2*1024*1024 {
		return setupcontrol.ErrInvalid
	}
	var release runtimeSetup.Release
	if json.Unmarshal(b, &release) != nil {
		return setupcontrol.ErrInvalid
	}
	driver, e := runtimeSetup.New(filepath.Join(root, "source"), release, nil)
	if e != nil {
		return e
	}
	userConfig, e := os.UserConfigDir()
	if e != nil {
		return e
	}
	state := filepath.Join(userConfig, "Shuomi", "setup", "v1")
	if e = privatefile.RejectPath(state); e != nil {
		return e
	}
	assets := filepath.Join(root, "ui")
	if _, e = os.Stat(filepath.Join(assets, "index.html")); e != nil {
		return e
	}
	listener, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		return e
	}
	defer listener.Close()
	code := setupcontrol.RandomSecret()
	handler := setupcontrol.NewHTTP(listener.Addr().String(), code, setupcontrol.New(state, driver), driver, http.FileServer(http.Dir(assets)))
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 * 1024}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	fmt.Printf("安装与模块管理：http://%s\n一次性配对码（仅在该页面输入）：%s\n本机回执目录：%s\n此权限不代表任何企业或业务用户身份。退出后需重新启动并配对。\n", listener.Addr(), code, state)
	e = server.Serve(listener)
	if e == http.ErrServerClosed {
		return nil
	}
	return e
}
