// Namespace for containing the builtin functions map.
package evaluator

import (
	"bytes"
	"fmt"
	"lisp/object"
	"strings"
)

// A map of all the built in functions in the interpreter
var builtins = map[string]object.Function{
	"+": func(env *object.Environment, args ...object.Object) object.Object {
		var result float64 = 0

		for _, arg := range args {
			switch obj := arg.(type) {
			case *object.Integer:
				result += float64(obj.Value)
			case *object.Float:
				result += obj.Value
			default:
				return badTypeError("+", obj)
			}
		}

		if isInt(result) {
			return &object.Integer{Value: int64(result)}
		}

		return &object.Float{Value: result}
	},
	"*": func(env *object.Environment, args ...object.Object) object.Object {
		var result float64 = 1

		for _, arg := range args {
			switch obj := arg.(type) {
			case *object.Integer:
				result *= float64(obj.Value)
			case *object.Float:
				result *= obj.Value
			default:
				return badTypeError("*", obj)
			}
		}

		if isInt(result) {
			return &object.Integer{Value: int64(result)}
		}
		return &object.Float{Value: result}
	},
	"-": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) == 0 {
			return noArgsError("-")
		}

		var nums []float64

		for _, arg := range args {
			switch obj := arg.(type) {
			case *object.Integer:
				nums = append(nums, float64(obj.Value))
			case *object.Float:
				nums = append(nums, obj.Value)
			default:
				return badTypeError("-", obj)
			}
		}

		if len(nums) == 1 {
			if isInt(nums[0]) {
				return &object.Integer{Value: -int64(nums[0])}
			}
			return &object.Float{Value: -nums[0]}
		} else {
			result := nums[0]

			for _, num := range nums[1:] {
				result -= num
			}

			if isInt(result) {
				return &object.Integer{Value: int64(result)}
			}
			return &object.Float{Value: result}
		}
	},
	"/": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) == 0 {
			return noArgsError("/")
		}

		var nums []float64

		for _, arg := range args {
			switch obj := arg.(type) {
			case *object.Integer:
				nums = append(nums, float64(obj.Value))
			case *object.Float:
				nums = append(nums, obj.Value)
			default:
				return badTypeError("/", obj)
			}
		}

		if len(nums) == 1 {
			return &object.Float{Value: 1 / nums[0]}
		} else {
			result := nums[0]

			for _, num := range nums[1:] {
				if num == 0 {
					return &object.ErrorObject{
						Error: "Attempted to divide by 0",
					}
				}
				result /= num
			}

			if isInt(result) {
				return &object.Integer{Value: int64(result)}
			}
			return &object.Float{Value: result}
		}
	},
	// Analogous to % in other languages like python, ruby, etc.
	"rem": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 2 {
			return wrongNumOfArgsError("rem", "2", len(args))
		}

		var ints [2]int64

		for i, arg := range args {
			switch obj := arg.(type) {
			case *object.Integer:
				ints[i] = obj.Value
			default:
				return badTypeError("/", obj)
			}

		}

		if ints[1] == 0 {
			return &object.ErrorObject{
				Error: "Attempted rem of 0",
			}
		}

		return &object.Integer{Value: ints[0] % ints[1]}
	},
	// Analogous to `==` in other languages, but with any amount of arguments
	"=": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) == 0 {
			return TRUE
		}

		obj := args[0]

		switch obj := obj.(type) {
		case *object.Integer:
			return numsEqual(float64(obj.Value), args[1:]...)
		case *object.Float:
			return numsEqual(obj.Value, args[1:]...)
		case *object.String:
			return stringsEqual(obj, args[1:]...)
		case *object.BooleanObject:
			return boolEqual(obj, args[1:]...)
		case *object.LambdaObject:
			return lambdasEqual(obj, args[1:]...)
		case *object.FunctionObject:
			return functionsEqual(obj, args[1:]...)
		default:
			return badTypeError("=", obj)
		}
	},
	"<": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) == 0 {
			return wrongNumOfArgsError("<", "at least 1", 0)
		}

		nums := []float64{}

		for _, arg := range args {
			switch arg := arg.(type) {
			case *object.Integer:
				nums = append(nums, float64(arg.Value))
			case *object.Float:
				nums = append(nums, arg.Value)
			default:
				return badTypeError("<", arg)
			}
		}

		for i, n := range nums[1:] {
			if n <= nums[i] {
				return FALSE
			}
		}

		return TRUE
	},
	">": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) == 0 {
			return wrongNumOfArgsError(">", "at least 1", 0)
		}

		nums := []float64{}

		for _, arg := range args {
			switch arg := arg.(type) {
			case *object.Integer:
				nums = append(nums, float64(arg.Value))
			case *object.Float:
				nums = append(nums, arg.Value)
			default:
				return badTypeError(">", arg)
			}
		}

		for i, n := range nums[1:] {
			if n >= nums[i] {
				return FALSE
			}
		}

		return TRUE
	},
	"not": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 1 {
			return wrongNumOfArgsError("not", "1", len(args))
		}

		if args[0].Type() == object.ERROR_OBJ {
			return args[0]
		}

		if evalTruthy(args[0]) {
			return FALSE
		}
		return TRUE
	},
	"and": func(env *object.Environment, args ...object.Object) object.Object {
		for _, arg := range args {
			if arg.Type() == object.ERROR_OBJ {
				return arg
			}

			if !evalTruthy(arg) {
				return FALSE
			}
		}

		return TRUE
	},
	"or": func(env *object.Environment, args ...object.Object) object.Object {
		for _, arg := range args {
			if arg.Type() == object.ERROR_OBJ {
				return arg
			}

			if evalTruthy(arg) {
				return TRUE
			}
		}

		return FALSE
	},
	// Construct a List Object from an argument list.
	"list": func(env *object.Environment, args ...object.Object) object.Object {
		return &object.List{
			Values: args,
		}
	},
	// Construct a Dictionary Object from an argument list.
	"dict": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args)%2 != 0 {
			return wrongNumOfArgsError("dict", "even number", len(args))
		}

		items := map[object.HashKey]object.DictPair{}

		for i := 0; i < len(args)-1; i += 2 {
			obj := args[i]
			value := args[i+1]

			key, ok := obj.(object.Hashable)

			if !ok {
				return badKeyError(obj)
			}

			items[key.HashKey()] = object.DictPair{
				Key:   obj,
				Value: value,
			}
		}

		return &object.Dictionary{
			Values: items,
		}
	},
	"first": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 1 {
			return wrongNumOfArgsError("first", "1", len(args))
		}

		if args[0].Type() != object.LIST_OBJ {
			return badTypeError("first", args[0])
		}

		list := args[0].(*object.List)
		return list.Values[0]
	},
	"rest": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 1 {
			return wrongNumOfArgsError("rest", "1", len(args))
		}

		if args[0].Type() != object.LIST_OBJ {
			return badTypeError("rest", args[0])
		}

		list := args[0].(*object.List)
		return &object.List{
			Values: list.Values[1:],
		}
	},
	"last": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 1 {
			return wrongNumOfArgsError("last", "1", len(args))
		}

		if args[0].Type() != object.LIST_OBJ {
			return badTypeError("last", args[0])
		}

		list := args[0].(*object.List)
		return list.Values[len(list.Values)-1]
	},
	"len": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 1 {
			return wrongNumOfArgsError("len", "1", len(args))
		}

		if args[0].Type() != object.LIST_OBJ {
			return badTypeError("len", args[0])
		}

		list := args[0].(*object.List)
		return &object.Integer{Value: int64(len(list.Values))}
	},
	// Takes two arguments, a list and an object.
	//
	// Returns a new list that is a copy of the given list, with
	// the object appended.
	"push": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 2 {
			return wrongNumOfArgsError("push", "2", len(args))
		}

		if args[0].Type() != object.LIST_OBJ {
			return &object.ErrorObject{
				Error: fmt.Sprintf("first argument to concat should be list. got=%T(%+v)",
					args[0], args[0]),
			}
		}

		list := args[0].(*object.List)

		newList := make([]object.Object, len(list.Values))
		copy(newList, list.Values)

		newList = append(newList, args[1])

		return &object.List{Values: newList}
	},
	// Takes two arguments, a list and an object.
	//
	// Appends the object to the list in place, returning
	// a reference to the same list.
	"push!": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 2 {
			return wrongNumOfArgsError("push", "2", len(args))
		}

		if args[0].Type() != object.LIST_OBJ {
			return &object.ErrorObject{
				Error: fmt.Sprintf("first argument to concat should be list. got=%T(%+v)",
					args[0], args[0]),
			}
		}

		list := args[0].(*object.List)
		list.Values = append(list.Values, args[1])

		return list
	},
	// Takes a list as its argument.
	//
	// Removes the last object from the list in place
	// and returns the popped object.
	"pop!": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 1 {
			return wrongNumOfArgsError("pop", "1", len(args))
		}

		if args[0].Type() != object.LIST_OBJ {
			return &object.ErrorObject{
				Error: fmt.Sprintf("argument to pop should be list. got=%T(%+v)",
					args[0], args[0]),
			}
		}

		list := args[0].(*object.List)

		if len(list.Values) == 0 {
			return &object.ErrorObject{
				Error: "attemped to pop from empty list",
			}
		}

		result := list.Values[len(list.Values)-1]

		list.Values = list.Values[:len(list.Values)-1]

		return result
	},
	// string representation of any object
	"str": func(env *object.Environment, args ...object.Object) object.Object {
		var result bytes.Buffer

		for _, arg := range args {
			result.WriteString(arg.Inspect())
		}

		return &object.String{
			Value: result.String(),
		}
	},
	"print": func(env *object.Environment, args ...object.Object) object.Object {
		objects := []string{}

		for _, arg := range args {
			objects = append(objects, arg.Inspect())
		}

		fmt.Println(strings.Join(objects, " "))

		return NULL
	},
	// Used to retrieve an item from a dictionary.
	//
	// `(get dict 'key')` is the equivalent of `dict['key']`
	// in other languages.
	"get": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 2 {
			wrongNumOfArgsError("get", "2", len(args))
		}

		dictObj := args[0]
		keyObj := args[1]

		if dictObj.Type() != object.DICT_OBJ {
			err := fmt.Sprintf("attempted to get from %s(%s) instead of dict", dictObj.Type(), dictObj.Inspect())
			return &object.ErrorObject{
				Error: err,
			}
		}
		dict := dictObj.(*object.Dictionary)

		key, ok := keyObj.(object.Hashable)
		if !ok {
			badKeyError(keyObj)
		}

		result, ok := dict.Values[key.HashKey()]

		if !ok {
			return NULL
		}

		return result.Value
	},
	// Used to add an item to a dictionary.
	//
	// `(set dict 'key' 5)` is the equivalent of `dict['key'] = 5`
	// in other languages.
	"set": func(env *object.Environment, args ...object.Object) object.Object {
		if len(args) != 3 {
			wrongNumOfArgsError("get", "3", len(args))
		}

		dictObj := args[0]
		keyObj := args[1]
		value := args[2]

		if dictObj.Type() != object.DICT_OBJ {
			err := fmt.Sprintf("attempted to get from %s(%s) instead of dict", dictObj.Type(), dictObj.Inspect())
			return &object.ErrorObject{
				Error: err,
			}
		}

		key, ok := keyObj.(object.Hashable)

		if !ok {
			badKeyError(keyObj)
		}

		dict := dictObj.(*object.Dictionary)
		dict.Values[key.HashKey()] = object.DictPair{
			Key:   keyObj,
			Value: value,
		}

		return dict
	},
}

func evalTruthy(obj object.Object) bool {
	if obj == NULL || obj == FALSE {
		return false
	}

	return true
}

func isInt(num float64) bool {
	return num == float64(int64(num))
}
