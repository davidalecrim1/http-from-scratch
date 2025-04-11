package fast

const (
	StatusOK                  int = 200
	StatusBadRequest          int = 400
	StatusNotFound            int = 404
	StatusInternalServerError int = 500
)

var StatusText = map[int]string{
	200: "OK",
	400: "Bad Request",
	404: "Not Found",
	500: "Internal Server Error",
}
