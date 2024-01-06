package commonhttp

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/konstantinfoerster/card-service-go/internal/common"
	"github.com/konstantinfoerster/card-service-go/internal/common/auth"
)

const HeaderHTMXRequest = "HX-Request"

func RenderLayout(c *fiber.Ctx, tmplName string, data fiber.Map) error {
	if data == nil {
		data = fiber.Map{}
	}

	user, _ := auth.UserFromCtx(c)

	data["User"] = NewClientUser(user)

	return c.Render(tmplName, data, "layouts/main")
}

func RenderPartial(c *fiber.Ctx, tmplName string, data any) error {
	if data == nil {
		data = fiber.Map{}
	}

	if mData, ok := data.(fiber.Map); ok {
		user, _ := auth.UserFromCtx(c)
		mData["User"] = NewClientUser(user)
	}

	return c.Render(tmplName, data)
}

func RenderJSON(c *fiber.Ctx, data interface{}) error {
	err := c.JSON(data)
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	return err
}

func IsHTMX(c *fiber.Ctx) bool {
	return strings.ToLower(c.Get(HeaderHTMXRequest)) == "true"
}

func AcceptsHTML(c *fiber.Ctx) bool {
	return strings.Contains(c.Get(fiber.HeaderAccept), fiber.MIMETextHTML)
}

func NewClientUser(u *auth.User) *ClientUser {
	if u == nil {
		return nil
	}

	username := u.Username
	if username == "" {
		username = "Unknown"
	}

	initials := []rune(username)[0:2]

	return &ClientUser{
		Username: username,
		Initials: string(initials),
	}
}

type ClientUser struct {
	Username string `json:"username"`
	Initials string `json:"initials"`
}

func DecodeBase64[T any](value string) (*T, error) {
	if strings.TrimSpace(value) == "" {
		return nil, common.NewInvalidInputMsg("unable-to-decode-value", "empty value")
	}

	rawJSON, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return nil, common.NewInvalidInputError(err, "unable-to-decode-value", "invalid encoding")
	}

	target := new(T)
	if err = json.Unmarshal(rawJSON, target); err != nil {
		return nil, err
	}

	return target, nil
}

func EncodeBase64[T any](value T) (string, error) {
	rawJwToken, err := json.Marshal(&value)
	if err != nil {
		return "", common.NewUnknownError(err, "unable-to-encode-value")
	}

	return base64.URLEncoding.EncodeToString(rawJwToken), nil
}
