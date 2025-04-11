package cors

import "fast"

func New() fast.Handler {
	return func(c *fast.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Method() == fast.MethodOptions {
			return c.SendStatus(fast.StatusNoContent)
		}

		return c.Next()
	}
}
