package web

import (
	"github.com/gofiber/fiber/v2"
)

type ClientUser struct {
	Username string `json:"username"`
	Initials string `json:"initials"`
}

func NewClientUser(u User) *ClientUser {
	if u.ID == "" {
		return nil
	}

	username := u.Email
	if username == "" {
		username = "Unknown"
	}

	initials := []rune(username)[0:2]

	return &ClientUser{
		Username: username,
		Initials: string(initials),
	}
}

func RenderPage(c *fiber.Ctx, cfg Auth, tmplName string, data fiber.Map) error {
	if data == nil {
		data = fiber.Map{}
	}

	user, _ := UserFromCtx(c)

	data["User"] = NewClientUser(user)
	data["activePage"] = tmplName
	data["LoginURL"] = cfg.LoginURL
	data["LogoutURL"] = cfg.LogoutURL

	return c.Render(tmplName, data, "layouts/main")
}

func RenderPartial(c *fiber.Ctx, tmplName string, data any) error {
	if data == nil {
		data = fiber.Map{}
	}

	if mData, ok := data.(fiber.Map); ok {
		user, _ := UserFromCtx(c)
		mData["User"] = NewClientUser(user)
		mData["activePage"] = tmplName
		mData["partial"] = true
	}

	return c.Render(tmplName, data)
}

func RenderJSON(c *fiber.Ctx, data any) error {
	err := c.JSON(data)
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	return err
}
