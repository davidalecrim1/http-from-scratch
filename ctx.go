package fast

type Ctx struct {
	Request  *Request
	Response Response
}

func (c *Ctx) SendString(body string) error {
	c.Response.SetBodyString(body)
	return nil
}
