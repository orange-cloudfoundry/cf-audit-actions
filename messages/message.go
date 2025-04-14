package messages

import (
	"fmt"
	"io"
	"os"

	"github.com/logrusorgru/aurora"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

var stdout = colorable.NewColorableStdout()

var C = aurora.NewAurora(isatty.IsTerminal(os.Stdout.Fd()))

var stopShow bool

func StopShow() bool {
	return stopShow
}

func SetStopShow(stShow bool) {
	stopShow = stShow
}

func Output() io.Writer {
	return stdout
}

func Println(a ...interface{}) (n int, err error) {
	if stopShow {
		return 0, nil
	}
	return fmt.Fprintln(stdout, a...)
}

func Print(a ...interface{}) (n int, err error) {
	if stopShow {
		return 0, nil
	}
	return fmt.Fprint(stdout, a...)
}

func Printf(format string, a ...interface{}) (n int, err error) {
	if stopShow {
		return 0, nil
	}
	return fmt.Fprintf(stdout, format, a...)
}

func Printfln(format string, a ...interface{}) (n int, err error) {
	if stopShow {
		return 0, nil
	}
	return fmt.Fprintf(stdout, format+"\n", a...)
}

func Error(str string) {
	if stopShow {
		return
	}
	_, err := Printfln("%s: %s", C.Red("Error"), str)
	if err != nil {
		fmt.Println("Error printing error message:", err)
	}
}

func Errorf(format string, a ...interface{}) {
	if stopShow {
		return
	}
	_, err := Printf("%s: ", C.Red("Error"))
	if err != nil {
		fmt.Println("Error printing error:", err)
	}
	_, err = Printfln(format, a...)
	if err != nil {
		fmt.Println("Error printing error format:", err)
	}
}

func Fatal(str string) {
	_, err := Printfln("%s: %s", C.Red("Error"), str)
	if err != nil {
		fmt.Println("Error printing fatal error message:", err)
	}
	os.Exit(1)
}

func Fatalf(format string, a ...interface{}) {
	_, err := Printf("%s: ", C.Red("Error"))
	if err != nil {
		fmt.Println("Error printing error:", err)
	}
	_, err = Printfln(format, a...)
	if err != nil {
		fmt.Println("Error printing fatal error format:", err)
	}
	os.Exit(1)
}

func Warning(str string) {
	if stopShow {
		return
	}
	_, err := Printfln("%s: %s", C.Magenta("Warning"), str)
	if err != nil {
		fmt.Println("Error printing warning message:", err)
	}
}

func Warningf(format string, a ...interface{}) {
	if stopShow {
		return
	}
	_, err := Printf("%s: ", C.Yellow("Warning"))
	if err != nil {
		fmt.Println("Error printing warning:", err)
	}
	_, err = Printfln(format, a...)
	if err != nil {
		fmt.Println("Error printing warning format:", err)
	}
}
