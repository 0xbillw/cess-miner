package utils

import (
	"bytes"
	"fmt"
	"runtime/debug"
)

func RecoverError(err interface{}) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%v\n", "--------------------panic--------------------")
	fmt.Fprintf(buf, "%v\n", err)
	fmt.Fprintf(buf, "%v\n", string(debug.Stack()))
	return buf.String()
}