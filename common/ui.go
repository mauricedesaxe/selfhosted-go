package common

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Renders a templ component and returns an error if any
// options can be passed to the component handler to modify the behavior
func RenderTempl(c *fiber.Ctx, component templ.Component, options ...func(*templ.ComponentHandler)) error {
	componentHandler := templ.Handler(component)
	for _, o := range options {
		o(componentHandler)
	}
	return adaptor.HTTPHandler(componentHandler)(c)
}

var Printer = message.NewPrinter(language.English)

func GetFileModTime(file string) time.Time {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return time.Time{}
	}
	return fileInfo.ModTime()
}

func Jsonify(data interface{}) string {
	stringified, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(stringified)
}

func Truncate(description string, wordLimit int) string {
	words := strings.Fields(description)
	if len(words) > wordLimit {
		return strings.Join(words[:wordLimit], " ")
	}
	return description
}
