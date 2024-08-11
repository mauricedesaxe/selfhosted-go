package marketing_test

import (
	"go-on-rails/cmd"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestRoutes(t *testing.T) {
	app := cmd.Run()

	Convey("GET / - should return home page", t, func() {
		req := httptest.NewRequest("GET", "/", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, resp.StatusCode, http.StatusOK)
		assert.Equal(t, resp.Header.Get("Content-Type"), "text/html; charset=utf-8")

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Contains(t, string(body), "<h1>Go on Rails</h1>")
	})
}
