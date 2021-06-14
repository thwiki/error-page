package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aerogo/aero"

	"github.com/thwiki/error-page/messages"
)

func checkStatus(t *testing.T, response *httptest.ResponseRecorder, expectedStatus int) {
	if response.Code != expectedStatus {
		t.Fatalf("Invalid status %d", response.Code)
	}
}

func checkContentType(t *testing.T, response *httptest.ResponseRecorder, expectedType string) {
	contentType := response.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, expectedType) {
		t.Fatalf("Invalid content type %s", contentType)
	}
}

func TestIndexRoutes(t *testing.T) {
	readConfig()
	AppConfig.Port = AppConfig.Port + 100
	messages.ReadMessages(AppConfig.Messages)

	app := configure(aero.New())
	request := httptest.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)

	checkStatus(t, response, http.StatusNotFound)
	checkContentType(t, response, "text/html")
}

func TestStaticStyleRoute(t *testing.T) {
	app := configure(aero.New())
	request := httptest.NewRequest("GET", AppConfig.Path+"/src/style.css", nil)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)

	checkStatus(t, response, http.StatusOK)
	checkContentType(t, response, "text/css")
}

func TestStaticFaviconRoute(t *testing.T) {
	app := configure(aero.New())
	request := httptest.NewRequest("GET", AppConfig.Path+"/src/favicon.ico", nil)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)

	checkStatus(t, response, http.StatusOK)
	checkContentType(t, response, "image/vnd.microsoft.icon")
}

func TestIndexRandom(t *testing.T) {
	length := len(messages.Messages)
	tries := 100000

	app := configure(aero.New())
	counts := make(map[string]int, length*2)
	for i := 0; i < tries; i++ {
		message := requestMessage(app)
		counts[message]++
	}

	counts2 := make([]int, length)

	for message, count := range counts {
		for i := 0; i < length; i++ {
			if messages.Messages[i].Text == message {
				counts2[i] = count
				break
			}
		}
	}

	var dev float64 = 0
	for i := 0; i < length; i++ {
		measuredCount := float64(counts2[i])
		expectedCount := float64(tries * messages.Messages[i].Type / messages.MaxRate)
		average := ((measuredCount + expectedCount) / 2)
		diff := (measuredCount / average) - (expectedCount / average)
		dev += diff * diff
	}

	averageDev := dev / float64(length)
	if averageDev > 0.1 {
		t.Fatalf("Deviate too much %f", averageDev)
	}
}

func requestMessage(app *aero.Application) string {
	request := httptest.NewRequest("GET", AppConfig.Path+"/503", nil)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)

	return getMessageFromBody(response.Body)
}

func getMessageFromBody(body *bytes.Buffer) string {
	html := body.String()
	lastIndex := strings.LastIndex(html, "<article>")
	html = html[lastIndex+9:]
	lastIndex = strings.Index(html, "</article>")
	return html[:lastIndex]
}
