// Command stalls is a scene that says where it is and never ends, for the
// golden package's test of a scene past its deadline.
package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("frame 3")
	time.Sleep(time.Hour)
}
