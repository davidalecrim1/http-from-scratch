package fast

const (
	StatusOK        = 200
	StatusNoContent = 204

	StatusBadRequest = 400
	StatusNotFound   = 404

	StatusInternalServerError = 500
	StatusServiceUnavailable  = 503
)

var StatusText = map[int]string{
	200: "OK",

	400: "Bad Request",
	404: "Not Found",

	500: "Internal Server Error",
}
