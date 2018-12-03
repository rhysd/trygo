package foo

import (
	"fmt"
)

func foo() (bool, error) {
	if true {
		var x, _err0 = fmt.Println("nested")
		if _err0 != nil {
			return false, _err0
		}
		switch x {
		case 42:
			n, _err0 := fmt.Println("a bit nested")
			if _err0 != nil {
				return false, _err0
			}
			for i := 0; i < 0; i++ {
				if _, err := fmt.Println("more nested"); err != nil {
					return false, err
				}
				for range []string{} {
					if _, err := fmt.Println("more more nested"); err != nil {
						return false, err
					}
					for {
						var _err0 error
						n, _err0 = fmt.Println("so nested")
						if _err0 != nil {
							return false, _err0
						}
						break
					}
				}
			}
			return n > 3, nil
		case 10:
			var ch chan int
			select {
			case <-ch:
				y, _err0 := fmt.Println("a bit nested")
				if _err0 != nil {
					return false, _err0
				}
				return y == 0, func() error {
					if _, err := fmt.Println("in func lit"); err != nil {
						return err
					}
					var _err0 error
					y, _err0 = fmt.Println("captured var")
					if _err0 != nil {
						return _err0
					}
					return nil
				}()
			}
		default:
		}
		if _, err := fmt.Println("nested"); err != nil {
			return false, err
		}
	}
	return true, nil
}
