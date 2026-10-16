package middleware

import (
	"github.com/gofiber/fiber/v2"
	"net/http/httptest"
	"testing"
)

func TestIdentityHeaderStrict(t *testing.T) {
	for _, value := range []string{"1junk", "-1", "0", "99999999999999999999999"} {
		app := fiber.New()
		app.Use(AuthMiddleware("local"))
		app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-User-ID", value)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 400 {
			t.Fatalf("%q got %d", value, resp.StatusCode)
		}
	}
}
func TestMiddlewareSeesFailureStatus(t *testing.T) {
	app := fiber.New()
	var seen int
	app.Use(func(c *fiber.Ctx) error { err := c.Next(); seen = ResponseStatus(c, err); return err })
	app.Get("/", func(c *fiber.Ctx) error { return fiber.ErrBadRequest })
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if seen != 400 {
		t.Fatalf("recorded %d", seen)
	}
}
