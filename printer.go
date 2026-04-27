package dispatch

type Printer interface {
	Print(...any)
	Printf(string, ...any)
}
