package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"go-on-rails/common"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var mailingQueue *common.Queue

func init() {
	mailingQueue = common.NewQueue(common.QueueOptions{})
	mailingQueue.StartJobQueue()
}

func AddRoutes(app *fiber.App) {
	auth := &AuthHandlers{}
	app.Get("/signup", auth.get_signup)
	app.Post("/signup", auth.post_signup)
	app.Get("/login", auth.get_login)
	app.Post("/login", auth.post_login)
	app.Get("/profile", auth.get_profile)
	app.Post("/change-password", auth.post_change_pass)
	app.Get("/forgot-password", auth.get_forgot_pass)
	app.Post("/forgot-password", auth.post_forgot_pass)
	app.Get("/reset-password", auth.get_reset_pass)
	app.Post("/reset-password", auth.post_reset_pass)
	app.Get("/logout", auth.get_logout)

	admin := &AdminHandlers{}
	app.Get("/admin", admin.get_admin)
	app.Post("/admin/smtp", admin.post_smtp)
	app.Get("/admin/users/:id", admin.get_user)
	app.Post("/admin/users/:id/reset-password", admin.post_reset_user_password)
	app.Get("/admin/signup-codes/new", admin.get_new_signup_code)
	app.Post("/admin/signup-codes", admin.post_signup_code)
	app.Post("/admin/signup-codes/delete", admin.delete_signup_codes)
	app.Post("/admin/signup-codes/delete/:code", admin.delete_signup_code)
	app.Get("/admin/signup-codes/:code", admin.get_edit_signup_code)
	app.Post("/admin/signup-codes/:code", admin.put_signup_code)
}

type AuthHandlers struct {
}

func (m *AuthHandlers) get_signup(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/login?error=Can't get session")
	}

	// redirect to the dashboard page if the user is already logged in
	userId := sess.Get("user_id")
	if userId != nil {
		return c.Redirect("/protected")
	}

	// render the signup page
	return common.RenderTempl(c, signup_page(Messages{
		Success: c.Query("success"),
		Error:   c.Query("error"),
	}))
}

func (m *AuthHandlers) post_signup(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/signup?error=Can't get session")
	}

	// redirect to the dashboard page if the user is already logged in
	userId := sess.Get("user_id")
	if userId != nil {
		return c.Redirect("/protected")
	}

	// get email and password from the form
	email := c.FormValue("email")
	password := c.FormValue("password")
	code := c.FormValue("code") // signup code

	// basic validation
	if email == "" || password == "" {
		return c.Redirect("/signup?error=Please enter your email and password")
	}
	if !strings.Contains(email, "@") {
		return c.Redirect("/signup?error=Please enter a valid email")
	}
	if len(password) < 6 {
		return c.Redirect("/signup?error=Password must be at least 6 characters")
	}
	if code == "" {
		return c.Redirect("/signup?error=Please enter your signup code")
	}

	// check if the email is already taken
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM users WHERE email = ?`, email)
	if err != nil {
		return c.Redirect("/signup?error=Email already taken")
	}

	// check if the signup code is valid
	var signupCode struct {
		Uses int `db:"uses"`
	}
	err = AuthDb.Get(&signupCode, `SELECT uses FROM signup_codes WHERE code = ? AND uses > 0`, strings.ToLower(code))
	if err != nil {
		return c.Redirect("/signup?error=Invalid signup code")
	}

	// hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return c.Redirect("/signup?error=Can't hash password")
	}

	// insert the user into the database
	_, err = AuthDb.Exec(`INSERT INTO users (email, password) VALUES (?, ?)`, email, string(hashedPassword))
	if err != nil {
		return c.Redirect("/signup?error=Can't insert user into database")
	}

	// if first user ever, give admin role
	if count == 0 {
		_, err = AuthDb.Exec(`INSERT INTO user_roles (user_id, role) VALUES ((SELECT id FROM users WHERE email = ?), "admin")`, email)
		if err != nil {
			return c.Redirect("/signup?error=Can't insert user role into database")
		}
	}

	// decrement the uses of the signup code
	_, err = AuthDb.Exec(`UPDATE signup_codes SET uses = uses - 1 WHERE code = ?`, strings.ToLower(code))
	if err != nil {
		return c.Redirect("/signup?error=Can't decrement signup code uses")
	}

	// save the user ID in the session
	var user struct {
		ID int `db:"id"`
	}
	err = AuthDb.Get(&user, `SELECT id FROM users WHERE email = ?`, email)
	if err != nil {
		return c.Redirect("/signup?error=Can't get user ID")
	}
	sess.Set("user_id", user.ID)
	err = sess.Save()
	if err != nil {
		return c.Redirect("/signup?error=Can't save user ID in session")
	}

	// redirect to the login page with a success message
	return c.Redirect(fmt.Sprintf("/login?success=Account created for %s. Please login", email))
}

func (m *AuthHandlers) get_login(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/login?error=Can't get session")
	}

	// redirect to the dashboard page if the user is already logged in
	userId := sess.Get("user_id")
	if userId != nil {
		return c.Redirect("/protected?error=You are already logged in")
	}

	// render the login page
	return common.RenderTempl(c, login_page(Messages{
		Success: c.Query("success"),
		Error:   c.Query("error"),
	}))
}

func (m *AuthHandlers) post_login(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/login?error=Can't get session")
	}

	// redirect to the dashboard page if the user is already logged in
	userId := sess.Get("user_id")
	if userId != nil {
		return c.Redirect("/protected?error=You are already logged in")
	}

	// get email and password from the form
	email := c.FormValue("email")
	password := c.FormValue("password")

	// basic validation
	if email == "" || password == "" {
		return c.Redirect("/login?error=Please enter your email and password")
	}
	if !strings.Contains(email, "@") {
		return c.Redirect("/login?error=Please enter a valid email")
	}
	if len(password) < 6 {
		return c.Redirect("/login?error=Password must be at least 6 characters")
	}

	// check if the user exists in the database
	type User struct {
		ID       int    `db:"id"`
		Password string `db:"password"`
	}
	var user User
	err = AuthDb.Get(&user, `SELECT id, password FROM users WHERE email = ?`, email)
	if err != nil {
		return c.Redirect("/login?error=Can't find user with the provided email")
	}

	// check if the password is correct
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return c.Redirect("/login?error=Incorrect password")
	}

	// save the user ID in the session
	sess, err = Store.Get(c)
	if err != nil {
		return c.Redirect("/login?error=Can't get session")
	}
	sess.Set("user_id", user.ID)
	err = sess.Save()
	if err != nil {
		return c.Redirect("/login?error=Can't save user ID in session")
	}

	// redirect to the dashboard page with a success message
	return c.Redirect("/protected?success=Logged in successfully")
}

func (m *AuthHandlers) get_profile(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/login?error=Please login to view your profile")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view your profile")
	}

	// get the user's metadata from the database
	var user UserMetadata
	err = AuthDb.Get(&user, `SELECT id, email, created_at FROM users WHERE id = ?`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user metadata")
	}

	// render the profile page
	return common.RenderTempl(c, profile_page(Messages{
		Success: c.Query("success"),
		Error:   c.Query("error"),
	}, user))
}

func (m *AuthHandlers) post_change_pass(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/login?error=Please login to view your profile")
	}

	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view your profile")
	}

	// get password and new password from the form
	password := c.FormValue("password")
	newPassword := c.FormValue("new-password")
	confirmPassword := c.FormValue("confirm-password")

	// basic validation
	if password == "" || newPassword == "" || confirmPassword == "" {
		return c.Redirect("/change-password?error=Please enter your password, new password, and confirm password")
	}
	if len(newPassword) < 6 {
		return c.Redirect("/change-password?error=New password must be at least 6 characters")
	}
	if newPassword != confirmPassword {
		return c.Redirect("/change-password?error=New password and confirm password do not match")
	}

	// get the user's password from the database
	var user struct {
		Password string `db:"password"`
	}
	err = AuthDb.Get(&user, `SELECT password FROM users WHERE id = ?`, userId.(int))
	if err != nil {
		return c.Redirect("/change-password?error=Can't get user password")
	}

	// check if the password is correct
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return c.Redirect("/change-password?error=Incorrect password")
	}

	// hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Redirect("/change-password?error=Can't hash password")
	}

	// update the user's password
	_, err = AuthDb.Exec(`UPDATE users SET password = ? WHERE id = ?`, string(hashedPassword), userId.(int))
	if err != nil {
		return c.Redirect("/change-password?error=Can't update user password")
	}

	// redirect to the profile page with a success message
	return c.Redirect("/profile?success=Password changed successfully")
}

func (m *AuthHandlers) get_forgot_pass(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/forgot-password?error=Can't get session")
	}

	// redirect to the dashboard page if the user is already logged in
	userId := sess.Get("user_id")
	if userId != nil {
		return c.Redirect("/protected")
	}

	// render the forgot password page
	return common.RenderTempl(c, forgot_password_page(Messages{
		Success: c.Query("success"),
		Error:   c.Query("error"),
	}))
}

func (m *AuthHandlers) post_forgot_pass(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/forgot-password?error=Can't get session")
	}

	// redirect to the dashboard page if the user is already logged in
	userId := sess.Get("user_id")
	if userId != nil {
		return c.Redirect("/protected")
	}

	// get email from the forms
	email := c.FormValue("email")
	if email == "" {
		return c.Redirect("/forgot-password?error=Please enter your email")
	}

	// check if the user exists in the database
	var user UserMetadata
	err = AuthDb.Get(&user, `SELECT id, email FROM users WHERE email = ?`, email)
	if err != nil {
		return c.Redirect("/forgot-password?error=Can't find user with the provided email")
	}

	// generate a unique token and save it in the database
	token := uuid.New().String()
	_, err = AuthDb.Exec(`INSERT INTO password_resets (user_id, token) VALUES (?, ?)`, user.ID, token)
	if err != nil {
		return c.Redirect("/forgot-password?error=Can't insert password reset token into database")
	}

	// Check if mailer is configured and send email with link to reset password
	if common.Mailer != nil && common.IsValidMailer(common.Mailer) {
		mailingQueue.AddJob(common.Job{
			Name: fmt.Sprintf("send-forgot-password-email-%s", email),
			Func: func() error {
				return common.Mailer.SendMail([]string{email}, "Password Reset", common.Env.BASE_URL+"/reset-password?token="+token)
			},
			Lockable: true, // don't want to send multiple emails at the same time to the same user
		})
	} else {
		return c.Redirect("/forgot-password?error=Can't send email because mailer is not configured, contact admin")
	}

	// redirect to the forgot password page with a success message
	return c.Redirect("/forgot-password?success=Check your email for a link to reset your password")
}

func (m *AuthHandlers) get_reset_pass(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/forgot-password?error=Can't get session")
	}

	// redirect to the dashboard page if the user is already logged in
	userId := sess.Get("user_id")
	if userId != nil {
		return c.Redirect("/protected")
	}

	// get the token from the query string
	token := c.Query("token")
	if token == "" {
		return c.Redirect("/forgot-password?error=Invalid token")
	}

	// check if the token exists in the database
	type PasswordReset struct {
		UserID int `db:"user_id"`
	}
	var passwordReset PasswordReset
	err = AuthDb.Get(&passwordReset, `SELECT user_id FROM password_resets WHERE token = ? AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)`, token)
	if err != nil {
		return c.Redirect("/forgot-password?error=Invalid token")
	}

	// render the reset password page
	return common.RenderTempl(c, reset_password_page(Messages{
		Success: c.Query("success"),
		Error:   c.Query("error"),
	}, token))
}

func (m *AuthHandlers) post_reset_pass(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/forgot-password?error=Can't get session")
	}

	// redirect to the dashboard page if the user is already logged in
	userId := sess.Get("user_id")
	if userId != nil {
		return c.Redirect("/protected?error=You are already logged in")
	}

	// get token and password from the form
	token := c.FormValue("token")
	password := c.FormValue("password")

	// basic validation
	if token == "" || password == "" {
		return c.Redirect("/reset-password?token=" + token + "&error=Please enter a password")
	}
	if len(password) < 6 {
		return c.Redirect("/reset-password?token=" + token + "&error=Password must be at least 6 characters")
	}

	// check if the token exists in the database
	var passwordReset struct {
		UserID int `db:"user_id"`
	}
	err = AuthDb.Get(&passwordReset, `SELECT user_id FROM password_resets WHERE token = ? AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)`, token)
	if err != nil {
		return c.Redirect("/reset-password?token=" + token + "&error=Invalid token")
	}

	// check if the new password is the same as the old password
	var user struct {
		Password string `db:"password"`
	}
	err = AuthDb.Get(&user, `SELECT password FROM users WHERE id = ?`, passwordReset.UserID)
	if err != nil {
		return c.Redirect("/reset-password?token=" + token + "&error=Can't get user password")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil {
		return c.Redirect("/reset-password?token=" + token + "&error=New password must be different from the old password")
	}

	// hash the new password and update the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return c.Redirect("/reset-password?token=" + token + "&error=Can't hash password")
	}
	_, err = AuthDb.Exec(`UPDATE users SET password = ? WHERE id = ?`, string(hashedPassword), passwordReset.UserID)
	if err != nil {
		return c.Redirect("/reset-password?token=" + token + "&error=Can't update user password")
	}

	// delete the token from the database
	_, err = AuthDb.Exec(`DELETE FROM password_resets WHERE token = ?`, token)
	if err != nil {
		return c.Redirect("/reset-password?token=" + token + "&error=Can't delete password reset token")
	}

	// redirect to the login page with a success message
	return c.Redirect("/login?success=Password reset successfully")
}

func (m *AuthHandlers) get_logout(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/?error=Can't get session")
	}

	// destroy the session
	sess.Destroy()

	// redirect to the login page with a success message
	return c.Redirect("/?success=Logged out successfully")
}

type AdminHandlers struct{}

func (m *AdminHandlers) get_admin(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// get user from db
	var me UserMetadata
	err = AuthDb.Get(&me, `SELECT id, email, created_at FROM users WHERE id = ?`, userId.(int))
	if err != nil {
		return common.RenderTempl(c, common.ErrorPage("💥 500", "Failed to get user metadata:", err.Error()))
	}

	// get all users from the database
	var users []UserMetadata
	err = AuthDb.Select(&users, `SELECT id, email, created_at FROM users`)
	if err != nil {
		return common.RenderTempl(c, common.ErrorPage("💥 500", "Failed to get users:", err.Error()))
	}

	// get all signup codes from the database
	var signupCodes []SignupCode
	err = AuthDb.Select(&signupCodes, `SELECT code, uses, created_at FROM signup_codes`)
	if err != nil {
		return common.RenderTempl(c, common.ErrorPage("💥 500", "Failed to get signup codes:", err.Error()))
	}

	// get SMTP settings
	var smtpSettings SMTPSettings
	err = common.MailDb.Get(&smtpSettings, `SELECT host, port, username, password FROM mailer_config`)
	if err != nil {
		if err == sql.ErrNoRows {
			smtpSettings = SMTPSettings{}
		} else {
			return common.RenderTempl(c, common.ErrorPage("💥 500", "Failed to get SMTP settings:", err.Error()))
		}
	}

	// render the admin page
	return common.RenderTempl(c, admin_page(admin_props{
		Me: me,
		Messages: Messages{
			Success: c.Query("success"),
			Error:   c.Query("error"),
		},
		Users:        users,
		SignupCodes:  signupCodes,
		SMTPSettings: smtpSettings,
	}))
}

func (m *AdminHandlers) post_smtp(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// validate the form
	host := c.FormValue("host")
	port := c.FormValue("port")
	username := c.FormValue("username")
	password := c.FormValue("password")
	intPort, err := strconv.Atoi(port)
	if err != nil {
		return c.Redirect("/admin?error=Invalid SMTP settings")
	}
	if !common.IsValidMailer(&common.MailerT{
		Host:     host,
		Port:     intPort,
		Username: username,
		Password: password,
	}) {
		return c.Redirect("/admin?error=Invalid SMTP settings")
	}

	// upsert SMTP settings
	_, err = common.MailDb.Exec(`
		INSERT INTO mailer_config (id, host, port, username, password) VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET 
		host = EXCLUDED.host, port = EXCLUDED.port, username = EXCLUDED.username, password = EXCLUDED.password`,
		host, intPort, username, password)
	if err != nil {
		return c.Redirect("/admin?error=Can't change SMTP settings because " + err.Error())
	}

	// redirect to the admin page with a success message
	return c.Redirect("/admin?success=SMTP settings updated successfully")
}

func (m *AdminHandlers) get_user(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/admin?error=Can't get session")
	}

	// redirect to the login page if the user is not logged in
	routeCallerUserId := sess.Get("user_id")
	if routeCallerUserId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, routeCallerUserId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// get user ID from params
	userId := c.Params("id")
	if userId == "" {
		return c.Redirect("/admin?error=Can't get user ID")
	}

	// get user from db
	var user UserMetadata
	err = AuthDb.Get(&user, `SELECT id, email, created_at FROM users WHERE id = ?`, userId)
	if err != nil {
		return c.Redirect("/admin?error=Can't get user metadata")
	}

	// render the user page
	return common.RenderTempl(c, user_page(user, Messages{
		Success: c.Query("success"),
		Error:   c.Query("error"),
	}, c.Query("new_password")))
}

func (m *AdminHandlers) post_reset_user_password(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/admin?error=Can't get session")
	}

	// redirect to the login page if the user is not logged in
	routeCallerUserId := sess.Get("user_id")
	if routeCallerUserId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, routeCallerUserId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// get user ID from params
	userId := c.Params("id")
	if userId == "" {
		return c.Redirect("/admin?error=Can't get user ID")
	}

	// get user from db
	var user UserMetadata
	err = AuthDb.Get(&user, `SELECT id, email, created_at FROM users WHERE id = ?`, userId)
	if err != nil {
		return c.Redirect("/admin?error=Can't get user metadata")
	}

	// generate a new random password of 33 characters
	byteArray := make([]byte, 33)
	_, err = rand.Read(byteArray)
	if err != nil {
		return c.Redirect("/admin?error=Can't generate new password")
	}
	newPassword := base64.URLEncoding.EncodeToString(byteArray)

	// hash the new password and update the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Redirect("/admin/users/" + userId + "?error=Can't hash password")
	}

	_, err = AuthDb.Exec(`UPDATE users SET password = ? WHERE id = ?`, string(hashedPassword), userId)
	if err != nil {
		return c.Redirect("/admin/users/" + userId + "?error=Can't update user password")
	}

	// redirect to the user page with a success message
	return c.Redirect("/admin/users/" + userId + "?success=Reset password successfully&new_password=" + newPassword)
}

func (m *AdminHandlers) get_new_signup_code(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/admin?error=Can't get session")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// render the new signup codes page
	return common.RenderTempl(c, new_signup_codes_page(Messages{
		Success: c.Query("success"),
		Error:   c.Query("error"),
	}))
}

func (m *AdminHandlers) post_signup_code(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/admin/signup-codes/new?error=Can't get session")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// basic validation
	code := c.FormValue("code")
	if code == "" {
		return c.Redirect("/admin/signup-codes/new?error=Please enter the code")
	}
	uses := c.FormValue("uses")
	if uses == "" {
		return c.Redirect("/admin/signup-codes/new?error=Please enter the number of uses")
	}
	usesInt, err := strconv.Atoi(uses)
	if err != nil {
		return c.Redirect("/admin/signup-codes/new?error=Invalid number of uses")
	}

	// replace spaces with hyphens and convert to lowercase
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ToLower(code)

	// insert the signup code into the database
	_, err = AuthDb.Exec(`INSERT INTO signup_codes (code, uses) VALUES (?, ?)`, code, usesInt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return c.Redirect("/admin/signup-codes/new?error=Signup code already exists")
		}
		return c.Redirect("/admin/signup-codes/new?error=Can't insert signup code into database")
	}

	// redirect to the new signup codes page with a success message
	return c.Redirect(fmt.Sprintf("/admin/signup-codes/%s?success=Generated signup code successfully", code))
}

func (m *AdminHandlers) get_edit_signup_code(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/admin?error=Can't get session")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// get the signup code from the URL
	code := c.Params("code")

	// get the signup code from the database
	var signupCode SignupCode
	err = AuthDb.Get(&signupCode, `SELECT code, uses, created_at FROM signup_codes WHERE code = ?`, code)
	if err != nil {
		return c.Redirect("/admin?error=Can't get signup code")
	}

	// render the edit signup code page
	return common.RenderTempl(c, edit_signup_codes_page(Messages{
		Success: c.Query("success"),
		Error:   c.Query("error"),
	}, signupCode))
}

func (m *AdminHandlers) put_signup_code(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/admin?error=Can't get session")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// get the signup code from the URL
	code := c.Params("code")

	// basic validation
	uses := c.FormValue("uses")
	if uses == "" {
		return c.Redirect(fmt.Sprintf("/admin/signup-codes/%s?error=Please enter the number of uses", code))
	}
	usesInt, err := strconv.Atoi(uses)
	if err != nil {
		return c.Redirect(fmt.Sprintf("/admin/signup-codes/%s?error=Invalid number of uses", code))
	}

	// update the signup code in the database
	_, err = AuthDb.Exec(`UPDATE signup_codes SET uses = ? WHERE code = ?`, usesInt, strings.ToLower(code))
	if err != nil {
		return c.Redirect(fmt.Sprintf("/admin/signup-codes/%s?error=Can't update signup code", code))
	}

	// redirect to the edit signup code page with a success message
	return c.Redirect(fmt.Sprintf("/admin/signup-codes/%s?success=Updated signup code successfully", code))
}

func (m *AdminHandlers) delete_signup_code(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/admin?error=Can't get session")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// get the signup code from the URL
	code := c.Params("code")

	// basic validation
	if code == "" {
		return c.Redirect("/admin?error=Please select a signup code to delete")
	}

	// delete the signup code from the database
	_, err = AuthDb.Exec(`DELETE FROM signup_codes WHERE code = ?`, code)
	if err != nil {
		return c.Redirect("/admin?error=Can't delete signup code")
	}

	// redirect to the admin page with a success message
	return c.Redirect("/admin?success=Deleted signup code successfully")
}

func (m *AdminHandlers) delete_signup_codes(c *fiber.Ctx) error {
	// get session
	sess, err := Store.Get(c)
	if err != nil {
		return c.Redirect("/admin?error=Can't get session")
	}

	// redirect to the login page if the user is not logged in
	userId := sess.Get("user_id")
	if userId == nil {
		return c.Redirect("/login?error=Please login to view the admin page")
	}

	// check if the user has the admin role
	var count int
	err = AuthDb.Get(&count, `SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = "admin"`, userId.(int))
	if err != nil {
		return c.Redirect("/login?error=Can't get user roles")
	}
	if count == 0 {
		return c.Redirect("/login?error=You do not have permission to view the admin page")
	}

	// get the signup code from the URL
	codesString := c.FormValue("codes")
	codes := strings.Split(codesString, ",")

	// basic validation
	if codesString == "" {
		return c.Redirect("/admin?error=Please select at least one signup code to delete")
	}
	if len(codes) == 0 || (len(codes) == 1 && codes[0] == "") {
		return c.Redirect("/admin?error=Please select at least one signup code to delete")
	}

	// delete the signup codes from the database
	for _, code := range codes {
		_, err = AuthDb.Exec(`DELETE FROM signup_codes WHERE code = ?`, strings.ToLower(code))
		if err != nil {
			return c.Redirect("/admin?error=Can't delete signup code: " + code)
		}
	}

	// redirect to the admin page with a success message
	return c.Redirect("/admin?success=Deleted " + strconv.Itoa(len(codes)) + " signup codes successfully")
}
