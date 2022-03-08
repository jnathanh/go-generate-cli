package climodel

// todo: can these be moved to the test definition file?

//go:generate go run ../../../generate/main.go -testsignature=Greet
func Greet(name string) (greeting string) {
	return "hello " + name + "\n"
}

//go:generate go run ../../../generate/main.go -testsignature=Dismiss
func Dismiss(name string) (greeting string) {
	return "goodbye " + name + "\n"
}
