## TODO
- [x] Finish the refactor and receive incoming request to trigger the proper handling.
- [x] Fix mess in the CORS tests when running all together.
- [x] Add a json format to the response.
- [x] Add persistant connections.
- [ ] Add grouping of routes.
- [ ] Add all other types of app.HTTP_METHOD -> https://gofiber.github.io/docs/api/app
- [ ] Decouple TCP from HTTP to have unit like tests.
- [ ] Add benchmarks to evaluate the data structure for the router.

### Example
func (app *App) Test(inputedReq *http.Request) (http.Response, error) {
	// create my internal request
	// send to the methods that do not rely on TCP
	// if needed, decouple the TCP and HTTP things.
}