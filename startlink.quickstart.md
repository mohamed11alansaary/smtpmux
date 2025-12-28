# Starlark Quick Start Guide (Go)

A practical, minimal guide to embedding **Starlark** as a scripting language in **Go**.

---

## What is Starlark?

- Python-like, deterministic scripting language
- Sandboxed by default (no I/O, no threads, no network)
- Used by Bazel, Tilt, Buck, Kubernetes tooling
- Ideal for **config, rules engines, plugins, policies**

Go package:

```
go.starlark.net/starlark
```

---

## Installation

```
go get go.starlark.net/starlark
```

---

## Smallest Working Example

### Go

```
package main

import (
	"fmt"
	"go.starlark.net/starlark"
)

func main() {
	thread := &starlark.Thread{Name: "main"}

	globals, err := starlark.ExecFile(
		thread,
		"example.star",
		`
x = 10
y = x * 2
`,
		nil,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(globals["y"]) // 20
}
```

**Notes**
- `ExecFile` runs a `.star` script (or inline source)
- Returns top-level variables as `globals`

---

## Calling a Starlark Function from Go

### script.star

```
def add(a, b):
    return a + b
```

### Go

```
globals, _ := starlark.ExecFile(thread, "script.star", nil, nil)

fn := globals["add"].(*starlark.Function)

result, err := starlark.Call(
	thread,
	fn,
	starlark.Tuple{
		starlark.MakeInt(2),
		starlark.MakeInt(3),
	},
	nil,
)
if err != nil {
	panic(err)
}

fmt.Println(result) // 5
```

---

## Exposing Go Functions to Starlark

### Go builtin

```
func hello(thread *starlark.Thread, fn *starlark.Builtin,
	args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {

	var name string
	if err := starlark.UnpackArgs(
		"hello", args, kwargs,
		"name", &name,
	); err != nil {
		return nil, err
	}

	return starlark.String("Hello " + name), nil
}
```

### Register builtin

```
globals := starlark.StringDict{
	"hello": starlark.NewBuiltin("hello", hello),
}

starlark.ExecFile(thread, "script.star", `
print(hello("Aman"))
`, globals)
```

---

## Passing Data from Go → Starlark

### Type mapping

| Go | Starlark |
|----|---------|
| int | starlark.Int |
| string | starlark.String |
| bool | starlark.Bool |
| []Value | starlark.List |
| map[string]Value | starlark.Dict |

### Example

```
globals := starlark.StringDict{
	"config": starlark.StringDict{
		"env":  starlark.String("prod"),
		"port": starlark.MakeInt(8080),
	},
}
```

Starlark usage:

```
print(config["env"])
```

---

## Reading Data from Starlark → Go

### Starlark

```
settings = {
    "host": "localhost",
    "port": 5432,
}
```

### Go

```
dict := globals["settings"].(*starlark.Dict)

portVal, _, _ := dict.Get(starlark.String("port"))
port := int(portVal.(starlark.Int).Int64())
```

---

## Modules and `load()`

### Go loader

```
thread := &starlark.Thread{
	Name: "main",
	Load: func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		if module == "math.star" {
			return starlark.StringDict{
				"pi": starlark.Float(3.14),
			}, nil
		}
		return nil, fmt.Errorf("unknown module: %s", module)
	},
}
```

### Starlark

```
load("math.star", "pi")
print(pi)
```

---

## Sandboxing and Safety

Starlark is safe **by default**:

- ❌ No filesystem access
- ❌ No networking
- ❌ No goroutines
- ❌ No `eval`

You must **explicitly expose**:
- Builtins
- Data
- Modules

### Execution limits

```
thread.SetMaxExecutionSteps(1_000_000)
```

---

## When to Use Starlark

### Good fit
- Rule engines
- Config more powerful than YAML
- Secure user scripting
- Plugin systems

### Not a good fit
- Async or concurrent tasks
- File/network-heavy scripts
- Full Python compatibility

---

## Minimal Project Structure

```
app/
├── main.go
└── rules.star
```

---

## TL;DR

1. `starlark.ExecFile()` → run scripts  
2. `starlark.NewBuiltin()` → expose Go APIs  
3. `starlark.Call()` → call Starlark functions  
4. Everything is sandboxed unless you allow it  

---

## References

- https://github.com/google/starlark-go
- https://bazel.build/rules/language
