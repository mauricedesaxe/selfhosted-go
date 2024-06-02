package marketing

import (
	"go-on-rails/auth"
	"go-on-rails/common"
	"time"

	"github.com/gofiber/fiber/v2"
)

func AddRoutes(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		common.SetCacheHeader(c, common.CacheOptions{
			MaxAge:               24 * time.Hour,
			StaleWhileRevalidate: 1 * time.Hour,
			StaleIfError:         1 * time.Hour,
		})
		return common.RenderTempl(c, home_page())
	})

	app.Get("/protected", func(c *fiber.Ctx) error {
		userId, err := auth.IsLoggedIn(c)
		if err != nil {
			return c.Redirect("/login?redirect=/protected&error=Please+log+in+to+view+this+page")
		}

		var email string
		err = auth.AuthDb.Get(&email, `SELECT email FROM users WHERE id = $1`, userId)
		if err != nil {
			return c.Redirect("/login?redirect=/protected&error=Please+log+in+to+view+this+page")
		}

		return common.RenderTempl(c, protected_page(email))
	})
}
