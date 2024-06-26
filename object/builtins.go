package object

import (
	"bytes"
	"fmt"
	"strings"
)

var TRUE = &BooleanObject{Value: true}
var FALSE = &BooleanObject{Value: false}
var NULL = &Null{}

// A map of all the built in functions in the interpreter

var Builtins = []*FunctionObject{
	{
		"+",
		func(args ...Object) Object {
			var result float64 = 0

			for _, arg := range args {
                num, ok := arg.(*Number)

                if !ok {
					return BadTypeError("+", arg)
                }

                result += num.Value
			}

			return &Number{Value: result}
		},
	},
	{
		"*",
		func(args ...Object) Object {
			var result float64 = 1

			for _, arg := range args {
                num, ok := arg.(*Number)

                if !ok {
					return BadTypeError("+", arg)
                }

                result *= num.Value
			}

			return &Number{Value: result}
		},
	},
	{
		"-",
		func(args ...Object) Object {
			if len(args) == 0 {
				return NoArgsError("-")
			}

			var nums []float64

			for _, arg := range args {
                num, ok := arg.(*Number)

                if !ok {
					return BadTypeError("+", arg)
                }

				nums = append(nums, num.Value)
			}

			if len(nums) == 1 {
				return &Number{Value: -nums[0]}
			} else {
				result := nums[0]

				for _, num := range nums[1:] {
					result -= num
				}

				return &Number{Value: result}
			}
		},
	},
	{
		"/",
		func(args ...Object) Object {
			if len(args) == 0 {
				return NoArgsError("/")
			}

			var nums []float64

			for _, arg := range args {
                num, ok := arg.(*Number)

                if !ok {
					return BadTypeError("/", num)
                }
                nums = append(nums, num.Value)
			}

			if len(nums) == 1 {
				return &Number{Value: 1 / nums[0]}
			} else {
				result := nums[0]

				for _, num := range nums[1:] {
					if num == 0 {
						return &ErrorObject{
							Error: "Attempted to divide by 0",
						}
					}
					result /= num
				}

				return &Number{Value: result}
			}
		},
	},
	// Analogous to % in other languages like python, ruby, etc.
	{
		"rem",
		func(args ...Object) Object {
			if len(args) != 2 {
				return WrongNumOfArgsError("rem", "2", len(args))
			}

            num, ok := args[0].(*Number)

            if !ok {
                return BadTypeError("rem", num)
            }

            top := num.Value

            num, ok = args[1].(*Number)

            if !ok {
                return BadTypeError("rem", num)
            }

            bottom := num.Value

			if bottom == 0 {
				return &ErrorObject{
					Error: "Attempted rem of 0",
				}
			}

            for (top >= bottom) {
                top -= bottom
            }

            return &Number{Value: top}
		},
	},
	// Analogous to `==` in other languages, but with any amount of arguments
	{
		"=",
		func(args ...Object) Object {
			if len(args) == 0 {
				return TRUE
			}

			obj := args[0]

			switch obj := obj.(type) {
			case *Number:
				return numsEqual(obj.Value, args[1:]...)
			case *String:
				return stringsEqual(obj, args[1:]...)
			case *BooleanObject:
				return boolEqual(obj, args[1:]...)
			case *LambdaObject:
				return lambdasEqual(obj, args[1:]...)
			case *FunctionObject:
				return functionsEqual(obj, args[1:]...)
			default:
				return BadTypeError("=", obj)
			}
		},
	},
	{
		"<",
		func(args ...Object) Object {
			if len(args) == 0 {
				return WrongNumOfArgsError("<", "at least 1", 0)
			}

			nums := []float64{}

			for _, arg := range args {
				switch arg := arg.(type) {
				case *Number:
					nums = append(nums, arg.Value)
				default:
					return BadTypeError("<", arg)
				}
			}

			for i, n := range nums[1:] {
				if n <= nums[i] {
					return FALSE
				}
			}

			return TRUE
		},
	},
	{
		">",
		func(args ...Object) Object {
			if len(args) == 0 {
				return WrongNumOfArgsError(">", "at least 1", 0)
			}

			nums := []float64{}

			for _, arg := range args {
				switch arg := arg.(type) {
				case *Number:
					nums = append(nums, arg.Value)
				default:
					return BadTypeError(">", arg)
				}
			}

			for i, n := range nums[1:] {
				if n >= nums[i] {
					return FALSE
				}
			}

			return TRUE
		},
	},
	{
		"not",
		func(args ...Object) Object {
			if len(args) != 1 {
				return WrongNumOfArgsError("not", "1", len(args))
			}

			if args[0].Type() == ERROR_OBJ {
				return args[0]
			}

			if evalTruthy(args[0]) {
				return FALSE
			}
			return TRUE
		},
	},
	{
		"and",
		func(args ...Object) Object {
			for _, arg := range args {
				if arg.Type() == ERROR_OBJ {
					return arg
				}

				if !evalTruthy(arg) {
					return FALSE
				}
			}

			return TRUE
		},
	},
	{
		"or",
		func(args ...Object) Object {
			for _, arg := range args {
				if arg.Type() == ERROR_OBJ {
					return arg
				}

				if evalTruthy(arg) {
					return TRUE
				}
			}

			return FALSE
		},
	},
	// Construct a List Object from an argument list.
	{
		"list",
		func(args ...Object) Object {
			values := make([]Object, len(args), len(args))

			// Loop ensures that the args are referenced as individual objects,
            // using args directly makes values a reference to args' underlying
            // slice, which can be changed elsewhere.
			for i, arg := range args {
				values[i] = arg
			}

			return &List{
				Values: values,
			}
		},
	},
	// Construct a Dictionary Object from an argument list.
	{
		"dict",
		func(args ...Object) Object {
			if len(args)%2 != 0 {
				return WrongNumOfArgsError("dict", "even number", len(args))
			}

			items := map[HashKey]DictPair{}

			for i := 0; i < len(args)-1; i += 2 {
				obj := args[i]
				value := args[i+1]

				key, ok := obj.(Hashable)

				if !ok {
					return BadKeyError(obj)
				}

				items[key.HashKey()] = DictPair{
					Key:   obj,
					Value: value,
				}
			}

			return &Dictionary{
				Values: items,
			}
		},
	},
	{
		"first",
		func(args ...Object) Object {
			if len(args) != 1 {
				return WrongNumOfArgsError("first", "1", len(args))
			}

			if args[0].Type() != LIST_OBJ {
				return BadTypeError("first", args[0])
			}

			list := args[0].(*List)

			if len(list.Values) == 0 {
				return NULL
			}

			return list.Values[0]
		},
	},
	{
		"rest",
		func(args ...Object) Object {
			if len(args) != 1 {
				return WrongNumOfArgsError("rest", "1", len(args))
			}

			if args[0].Type() != LIST_OBJ {
				return BadTypeError("rest", args[0])
			}

			list := args[0].(*List)

			if len(list.Values) == 0 {
				return NULL
			}

			return &List{
				Values: list.Values[1:],
			}
		},
	},
	{
		"last",
		func(args ...Object) Object {
			if len(args) != 1 {
				return WrongNumOfArgsError("last", "1", len(args))
			}

			if args[0].Type() != LIST_OBJ {
				return BadTypeError("last", args[0])
			}

			list := args[0].(*List)

			if len(list.Values) == 0 {
				return NULL
			}

			return list.Values[len(list.Values)-1]
		},
	},
	{
		"len",
		func(args ...Object) Object {
			if len(args) != 1 {
				return WrongNumOfArgsError("len", "1", len(args))
			}

			switch args[0].Type() {
			case LIST_OBJ:
				list := args[0].(*List)
				return &Number{Value: float64(len(list.Values))}
			case STRING_OBJ:
				str := args[0].(*String)
				return &Number{Value: float64(len(str.Value))}
			default:
				return BadTypeError("len", args[0])
			}
		},
	},
	// Takes two arguments, a list and an
	//
	// Returns a new list that is a copy of the given list, with
	// the object appended.
	{
		"push",
		func(args ...Object) Object {
			if len(args) != 2 {
				return WrongNumOfArgsError("push", "2", len(args))
			}

			if args[0].Type() != LIST_OBJ {
				return &ErrorObject{
					Error: fmt.Sprintf("first argument to push should be list. got=%T(%+v)",
						args[0], args[0]),
				}
			}

			list := args[0].(*List)

			newList := make([]Object, len(list.Values))
			copy(newList, list.Values)

			newList = append(newList, args[1])

			return &List{Values: newList}
		},
	},
	// string representation of any object
	{
		"str",
		func(args ...Object) Object {
			var result bytes.Buffer

			for _, arg := range args {
				result.WriteString(arg.Inspect())
			}

			return &String{
				Value: result.String(),
			}
		},
	},
	{
		"print",
		func(args ...Object) Object {
			objects := []string{}

			for _, arg := range args {
				objects = append(objects, arg.Inspect())
			}

			fmt.Println(strings.Join(objects, " "))

			return NULL
		},
	},
	// Used to retrieve an item from a dictionary.
	//
	// `(get dict 'key')` is the equivalent of `dict['key']`
	// in other languages.
	{
		"get",
		func(args ...Object) Object {
			if len(args) != 2 {
				WrongNumOfArgsError("get", "2", len(args))
			}

			dictObj := args[0]
			keyObj := args[1]

			if dictObj.Type() != DICT_OBJ {
				err := fmt.Sprintf("attempted to get from %s(%s) instead of dict", dictObj.Type(), dictObj.Inspect())
				return &ErrorObject{
					Error: err,
				}
			}
			dict := dictObj.(*Dictionary)

			key, ok := keyObj.(Hashable)
			if !ok {
				BadKeyError(keyObj)
			}

			result, ok := dict.Values[key.HashKey()]

			if !ok {
				return NULL
			}

			return result.Value
		},
	},
	// Used to add an item to a dictionary.
	//
	// `(set dict 'key' 5)` is the equivalent of `dict['key'] = 5`
	// in other languages.
	{
		"set",
		func(args ...Object) Object {
			if len(args) != 3 {
				WrongNumOfArgsError("get", "3", len(args))
			}

			dictObj := args[0]
			keyObj := args[1]
			value := args[2]

			if dictObj.Type() != DICT_OBJ {
				err := fmt.Sprintf("attempted to get from %s(%s) instead of dict", dictObj.Type(), dictObj.Inspect())
				return &ErrorObject{
					Error: err,
				}
			}

			key, ok := keyObj.(Hashable)

			if !ok {
				BadKeyError(keyObj)
			}

			dict := dictObj.(*Dictionary)
			dict.Values[key.HashKey()] = DictPair{
				Key:   keyObj,
				Value: value,
			}

			return dict
		},
	},
}

func GetBuiltinByName(name string) *FunctionObject {
	for _, builtin := range Builtins {
		if builtin.Name == name {
			return builtin
		}
	}

	return nil
}

func evalTruthy(obj Object) bool {
	if b, ok := obj.(*BooleanObject); ok {
		return b.Value
	}

	if _, ok := obj.(*Null); ok {
		return false
	}

	return true
}

func isInt(num float64) bool {
	return num == float64(int64(num))
}
