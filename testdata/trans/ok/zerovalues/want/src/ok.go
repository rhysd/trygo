package foo

import (
	"fmt"
	"unsafe"
)

type S struct {
	i int
}

type I interface {
}

type MyInt int

func Foo() (int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, complex64, complex128, string, unsafe.Pointer, rune, []int, [1]int, *int, func(), interface{}, I, MyInt, map[int]int, chan int, struct{ i int }, byte, error) {
	n, _err0 := fmt.Println("wow...")
	if _err0 != nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.0, 0.0, 0i, 0i, "", nil, 0, nil, [1]int{}, nil, nil, nil, nil, 0, nil, nil, struct{ i int }{}, 0, _err0
	}
	var _err1 error
	n, _err1 = fmt.Println("so long...")
	if _err1 != nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.0, 0.0, 0i, 0i, "", nil, 0, nil, [1]int{}, nil, nil, nil, nil, 0, nil, nil, struct{ i int }{}, 0, _err1
	}
	if _, err := fmt.Println("too long......"); err != nil {
		return 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0.0, 0.0, 0i, 0i, "", nil, 0, nil, [1]int{}, nil, nil, nil, nil, 0, nil, nil, struct{ i int }{}, 0, err
	}
	return 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 1.0, 2.0, 1i, 2i, "foo", unsafe.Pointer(&n), 'a', []int{}, [1]int{1}, &n, func() {}, &n, &n, MyInt(1), map[int]int{}, make(chan int), struct{ i int }{1}, 0, nil
}

func Bar() error {
	if _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, err := Foo(); err != nil {
		return err
	}
	return nil
}
