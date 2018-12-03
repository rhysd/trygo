package foo

import (
	"fmt"
)

func foo() (bool, error) {
	if true {
		var x = try(fmt.Println("nested"))
		switch x {
		case 42:
			n := try(fmt.Println("a bit nested"))
			for i := 0; i < 0; i++ {
				try(fmt.Println("more nested"))
				for range []string{} {
					try(fmt.Println("more more nested"))
					for {
						n = try(fmt.Println("so nested"))
						break
					}
				}
			}
			return n > 3, nil
		case 10:
			var ch chan int
			select {
			case <-ch:
				y := try(fmt.Println("a bit nested"))
				return y == 0, func() error {
					try(fmt.Println("in func lit"))
					y = try(fmt.Println("captured var"))
					return nil
				}()
			}
		default:
		}
		try(fmt.Println("nested"))
	}
	return true, nil
}
