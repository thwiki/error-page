package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"gopkg.in/yaml.v2"

	"github.com/aerogo/aero"

	"github.com/thwiki/error-page/messages"

	"github.com/thwiki/error-page/components"
)

type Config struct {
	Port     int                  `yaml:"port"`
	GZip     bool                 `yaml:"gzip"`
	Images   string               `yaml:"images"`
	Messages string               `yaml:"messages"`
	Path     string               `yaml:"path"`
	Status   map[int]StatusConfig `yaml:"status,omitempty"`
}

type StatusConfig struct {
	Title   string   `yaml:"title"`
	Message bool     `yaml:"message"`
	Class   string   `yaml:"class"`
	Main    []string `yaml:"main"`
}

var AppConfig Config

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-c
		cleanup()
		os.Exit(0)
	}()

	readConfig()
	go messages.ReadMessages(AppConfig.Messages)
	go messages.WatchMessages(AppConfig.Messages)

	app := aero.New()
	// AppConfig.Port = AppConfig.Port+100
	app.Config.GZip = AppConfig.GZip
	app.Config.Ports.HTTP = AppConfig.Port

	configure(app).Run()
}

func configure(app *aero.Application) *aero.Application {
	app.Rewrite(func(ctx aero.RewriteContext) {
		path := ctx.Path()

		if strings.HasPrefix(path, AppConfig.Path) && !strings.HasSuffix(path, "/") {
			return
		}
		ctx.SetPath(AppConfig.Path + "/404")
	})

	app.Get(AppConfig.Path+"/src/*file", func(ctx aero.Context) error {
		return ctx.File(AppConfig.Images + ctx.Get("file"))
	})

	app.Get(AppConfig.Path+"/*status", func(ctx aero.Context) error {
		//host := ctx.Request().Host()
		//fmt.Println(host)
		statusText := ctx.Get("status")

		status, err := strconv.Atoi(statusText)
		if err != nil || http.StatusText(status) == "" {
			status = http.StatusNotFound
		}
		statusConfig, ok := AppConfig.Status[status]
		if !ok {
			status = http.StatusNotFound
			statusConfig, ok = AppConfig.Status[status]
			if !ok {
				statusConfig = StatusConfig{}
			}
		}

		message := messages.EmptyMessage
		if statusConfig.Message {
			message = messages.RandomMessage()
		}

		response := ctx.Response()
		response.SetHeader("Content-Security-Policy", "default-src 'self'; script-src 'none'")
		response.SetHeader("X-Content-Type-Options", "nosniff")
		response.SetHeader("X-Frame-Options", "SAMEORIGIN")
		response.SetHeader("Referrer-Policy", "same-origin")
		response.SetHeader("Feature-Policy", "autoplay 'self'; sync-xhr 'self'; microphone 'self'")
		response.SetHeader("X-XSS-Protection", "1; mode=block")

		ctx.SetStatus(status)
		return ctx.HTML(components.Layout(AppConfig.Path, statusConfig.Title, statusConfig.Class, statusConfig.Main, message.Text))
	})

	return app
}

func cleanup() {
	fmt.Println("cleaning up")
	messages.UnwatchMessages()
	fmt.Println("finish clean up")
}

func readConfig() {
	file, err := ioutil.ReadFile("/etc/error-page/config.yml")
	if err != nil {
		panic(err)
	}
	yamlString := string(file)

	err = yaml.Unmarshal([]byte(yamlString), &AppConfig)
	if err != nil {
		panic(err)
	}

	fmt.Println(AppConfig)
}
