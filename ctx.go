package fast

type Ctx struct {
	Request  *Request
	Response *Response
	index    int
	handlers []Handler
}

func (c *Ctx) SendString(body string) error {
	c.Response.SetBodyString(body)
	return nil
}

func (c *Ctx) Set(key, val string) {
	c.Response.AddHeader(key, val)
}

func (c *Ctx) Method() string {
	return c.Request.Method
}

func (c *Ctx) SendStatus(status int) error {
	c.Status(status)
	return nil
}

func (c *Ctx) Status(status int) *Ctx {
	c.Response.SetStatus(status)
	return c
}

func (c *Ctx) Next() error {
	c.index++
	if !(c.index < len(c.handlers)) {
		panic("index is over the handlers in the context")
	}

	return c.handlers[c.index](c)
}
