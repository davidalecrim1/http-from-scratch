package fast

type Router interface {
	Add(method string, path string, handlers ...Handler) Router
}
