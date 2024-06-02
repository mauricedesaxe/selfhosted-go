package auth

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Checks if user is logged in. Returns the user ID if logged in, otherwise returns an error.
func IsLoggedIn(c *fiber.Ctx) (int, error) {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return 0, fmt.Errorf("failed to get session: %v", err)
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return 0, fmt.Errorf("user is not logged in")
	}

	return userId.(int), nil
}
