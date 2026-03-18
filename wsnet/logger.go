package wsnet

import "log"

var Verbose bool

func logVerbose(v ...interface{}) {
	if Verbose {
		log.Println(v...)
	}
}

func logVerbosef(format string, v ...interface{}) {
	if Verbose {
		log.Printf(format, v...)
	}
}
