package web

import (
	"github.com/gofiber/fiber/v2"
)

func RenderPage(c *fiber.Ctx, tmplName string, data fiber.Map) error {
	if data == nil {
		data = fiber.Map{}
	}

	user, _ := UserFromCtx(c)

	data["User"] = NewClientUser(user)
	data["activePage"] = tmplName

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

func RenderJSON(c *fiber.Ctx, data interface{}) error {
	err := c.JSON(data)
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	return err
}
