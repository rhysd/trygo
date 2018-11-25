TryGo: Go with 'try' operator
=============================

This is a translator of 'TryGo' as my experiment that what happens if Go were having 'try' function.
Basic idea of `try()` came from Rust's `try!` macro (or `?` operator). `try()` handles `if err != nil`
check implicitly.

Go:

```go
func CreateFile(subdir, filename string, content []byte) error {
    cwd, err := os.Getwd()
    if err != nil {
        return err
    }

    if err := os.Mkdir(filepath.Join(cwd, subdir)); err != nil {
        return err
    }

    f, err := os.Create(filepath.Join(cwd, subdir, filename))
    if err != nil {
        return err
    }
    defer f.Close()

    if _, err := f.Write(content); err != nil {
        return err
    }
    return nil
}
```

TryGo:

```go
func CreateFile(subdir, filename string, content []byte) error {
    cwd := try(os.Getwd())
    try(os.Mkdir(filepath.Join(cwd, subdir)))
    f := try(os.Create(filepath.Join(cwd, subdir, filename)))
    defer f.Close()
    try(f.Write(content))
    return nil
}
```

There is only one difference between Go and TryGo. Special magic function `try()` is provided in TryGo.

## Spec

`try` looks function, but actually it is a special operator. It has variadic parameters and variadic
return values. In terms of Go, `try` looks like:

```
func try(ret... interface{}, err error) (... interface{})
```

Actually `try()` is a set of macros which takes one function call and expands it to a code with error
check. It takes one function call as argument since Go only allows multiple values as return values
of function call.

In following subsections, `$zerovals` is expanded to zero-values of return values of the function.
For example, when `try()` is used in `func () (int, error)`, `$zerovals` will be `0`. When it is used
in `func () (*SomeStruct, SomeInterface, SomeStruct, error)`, `$zerovals` will be `nil, nil, SomeStruct{}`.

### Definition statement

```
$Vars := try($CallExpr)

var $Vars = try($CallExpr)
```

Expanded to:

```
$Vars, err := $CallExpr
if err != nil {
    return $zerovals, err
}

var $Vars, err = $CallExpr
if err != nil {
    return $zerovals, err
}
```

### Assignment statement

```
$Assignee = try($CallExpr)
```

Expanded to:

```
$Assignee, err = $CallExpr
if err != nil {
    return $zerovals, err
}
```

### Call statement

```
try($CallExpr)
```

Expanded to:

```
if $underscores, err := $CallExpr; err != nil {
    return err
}
```

`$underscores,` is a set of `_`s which ignores all return values from `$CallExpr`. For example, when
calling `func() (int, error)`, it is expanded to `_`. When calling `func() (A, B, error)`, it is expanded
to `_, _`. When calling `func() error`, it is expanded to an empty.

### Expression

```
try($CallExpr)
```

Expanded to:

```
if err := $CallExpr; err != nil
if err != nil {
    return $zerovals, err
}
```

This should be nested. For example,

```
x := try(Foo(try($CallExpr), arg))
```

```
tmp, err := $CallExpr
if err != nil {
    return $zerovals, err
}
x, err := Foo(tmp, arg)
if err != nil {
    return $zerovals, err
}
```

### Ill-formed cases

- `try()` cannot take other than function call. For example, `try(42)` is ill-formed.
- `try()` is expanded to code including `return`. Using it outside functions is ill-formed.
- When function called in `try()` invocation does not return `error` as last of return values, it is ill-formed.

These ill-formed code should be detected by translator and it will raise an error.

## Why `try()` 'function'? Why not `?` operator?

Following code may look even better. At least I think so.

```go
func CreateFile(subdir, filename string, content []byte) error {
    cwd := os.Getwd()?
    os.Mkdir(filepath.Join(cwd, subdir))?
    f := os.Create(filepath.Join(cwd, subdir, filename))?
    defer f.Close()
    f.Write(content)?
    return nil
}
```

The reason why I adopted `try()` function is that ...TODO

## Installation

Download an executable binary from [release page](https://github.com/rhysd/trygo/releases) (NOT YET).

To build from source:

```
$ go get -u github.com/rhysd/trygo/cmd/trygo
```

## License

[MIT License](LICENSE.txt)

TODO: まずは $retvals, err := Foo(...) に変換したあと，後ろの if err != nil を入れないでおく．次に型チェックして $retvals の型を得る．再度 AST をトラバースして該当箇所の型情報を手に入れ，if err != nil を補完する
