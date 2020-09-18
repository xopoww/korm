package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

type authChecker struct {
}

func (a * authChecker) Check(r * http.Request)(bool, error) {
	err := checkAuthCookie(r)
	switch {
	case err == nil:
		aaLogger.Debugf("allowed an api call from %s", r.Host)
		return true, nil
	case errors.Is(err, http.ErrNoCookie):
		aaLogger.Debugf("denied an api call from %s", r.Host)
		return false, nil
	default:
		return false, err
	}
}

func setApiSubroutes (s *mux.Router) {
	pool := NewMethodPool(&authChecker{})

	mAddDish := NewMethod(func(list *apiParamList)error{
		name := list.str["name"]
		description := list.str["description"]
		quantity := list.num["quantity"]

		_, err := addDish(name, description, quantity)
		if err != nil {
			return &ErrInternal{err}
		}

		return nil
	})
	mAddDish.AddParam("name", ParamTypeSingle, true)
	mAddDish.AddParam("description", ParamTypeSingle, false)
	mAddDish.AddParam("quantity", ParamTypeSingle | ParamTypeNumerical, true)
	pool.AddMethod("add_dish", mAddDish)

	mSubDish := NewMethod(func(list *apiParamList)error{
		id := list.num["id"]
		delta := list.num["delta"]

		err := subDish(id, delta)

		var bid *ErrBadID
		var bag *ErrBadArgument
		switch {
		case err == nil:
			break
		case errors.As(err, &bid) || errors.As(err, &bag):
			return err
		default:
			return &ErrInternal{err}
		}
		return nil
	})
	mSubDish.AddParam("id", ParamTypeNumerical | ParamTypeSingle, true)
	mSubDish.AddParam("delta", ParamTypeNumerical | ParamTypeSingle, true)
	pool.AddMethod("sub_dish", mSubDish)

	s.Handle("/{method}", pool)
	return
}

// Error wrapper for distinguishing server fault from client fault
type ErrInternal struct {
	Err error
}

func (e * ErrInternal) Error() string {
	return "internal error: " + e.Err.Error()
}

func (e * ErrInternal) Unwrap() error {
	return e.Err
}

// interface that checks by request if the client is allowed to make a request
type Checker interface{
	Check(*http.Request)(allow bool, err error)
}

type MethodPool struct {
	checker		Checker
	methods		map[string]apiMethod
}

func NewMethodPool(checker Checker) *MethodPool {
	return &MethodPool{
		checker: checker,
		methods: make(map[string]apiMethod),
	}
}

func (m * MethodPool) AddMethod(name string, method *apiMethod) {
	m.methods[name] = *method
}

func (m * MethodPool) ServeHTTP(w http.ResponseWriter, r * http.Request) {
	// perform the check before processing the request
	allow, err := m.checker.Check(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !allow {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// try to find the requested method
	methodName := mux.Vars(r)["method"]
	if method, found := m.methods[methodName]; found {
		// method exists - execute it...
		err := method.Execute(r)
		// ...and examine the error
		var ie *ErrInternal
		switch {
		case err == nil:
			// everything went ok
			w.WriteHeader(http.StatusOK)
		case errors.As(err, &ie):
			// internal error - server fault
			http.Error(w, err.Error(), http.StatusInternalServerError)
		default:
			// other error - client fault
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	} else {
		// method does not exist
		http.Error(w, "no such method: "+methodName, http.StatusNotFound)
	}
}

/*
Bitmask constants for apiParam.valueType

If ParamTypeNumerical is set to true, any values that cannot
be converted to int using strconv.ParseInt will cause an error.

If ParamTypeSingle is not set, a single value will cause an error.

If ParamTypeArray is not set, an array of values will cause an error.

If both ParamTypeSingle and ParamTypeArray are set, both single
and multiple values are accepted.

*/
const (
	ParamTypeNumerical = 1
	ParamTypeSingle = 1 << 1
	ParamTypeArray = 1 << 2
)

/* Represents an api parameter
valueType is a bitmask of different properties described in the constants above
 */
type apiParam struct {
	name				string
	valueType			uint
}

/* Represents api method
*/
type apiMethod struct {
	requiredParams		[]apiParam
	optionalParams		[]apiParam
	action				func(list * apiParamList)error
}

func NewMethod(action func(*apiParamList)error) *apiMethod {
	return &apiMethod{
		requiredParams: make([]apiParam, 0),
		optionalParams: make([]apiParam, 0),
		action: action,
	}
}

func (m * apiMethod) AddParam(name string, valueType uint, required bool) {
	if required {
		m.requiredParams = append(m.requiredParams, apiParam{
			name:	name,
			valueType: valueType,
		})
	} else {
		m.optionalParams = append(m.optionalParams, apiParam{
			name:	name,
			valueType: valueType,
		})
	}
}
/* Execute api method requested by r
Checks for all required params in URL Query values of r
(returns *ErrParamMissing if one or more required params are missing of
*ErrParamType if conversion of one or more params (required or optional) to
expected type fails). */
func (m * apiMethod) Execute(r * http.Request)error {
	err := r.ParseForm()
	if err != nil {
		return errors.New(fmt.Sprintf("parse form: %s", err))
	}

	params := makeApiParamList()
	for _, param := range m.requiredParams {
		value := r.Form[param.name]
		err := params.Add(param, value, true)
		if err != nil {
			return err
		}
	}
	for _, param := range m.optionalParams {
		value := r.Form[param.name]
		err := params.Add(param, value, false)
		if err != nil {
			return err
		}
	}

	return m.action(params)
}

type ErrParamMissing struct {
	paramName		string
}

func (e * ErrParamMissing) Error() string {
	return fmt.Sprintf("required parameter missing: %s", e.paramName)
}

type ErrParamType struct {
	param			apiParam
}

func (e * ErrParamType) Error() string {
	var typeString string
	switch t := e.param.valueType; {
	case t & ParamTypeSingle == 0:
		typeString = "array of "
	case t & ParamTypeArray == 0:
		typeString = "single value of "
	default:
		typeString = "array/single value of "
	}
	if e.param.valueType & ParamTypeNumerical != 0 {
		typeString += "integer"
	} else {
		typeString += "string"
	}

	return fmt.Sprintf("parameter %s must be of type %s", e.param.name, typeString)
}

// convenience struct to store api parameters separated by their types
type apiParamList struct {
	num		map[string]int
	nums	map[string][]int
	str		map[string]string
	strs	map[string][]string
}

/* Check if value (from url.Values[key]) satisfies param type and add it
*/
func (a * apiParamList) Add(param apiParam, value []string, required bool)error {
	switch len(value){
	case 0:
		// param missing
		if required {
			return &ErrParamMissing{param.name}
		}
		return nil
	case 1:
		// single param
		if param.valueType & ParamTypeSingle == 0{
			// must be an array
			return &ErrParamType{param}
		}

		if param.valueType & ParamTypeNumerical != 0 {
			// must be int
			valNum, err := strconv.ParseInt(value[0], 10, 0)
			if err != nil {
				return &ErrParamType{param}
			}
			a.num[param.name] = int(valNum)
			return nil

		} else {
			// must be string
			a.str[param.name] = value[0]
			return nil
		}
	default:
		// array
		if param.valueType & ParamTypeArray == 0 {
			// must be single value
			return &ErrParamType{param}
		}

		if param.valueType & ParamTypeNumerical != 0 {
			// must be array of int
			valNums := make([]int, len(value))
			for index, val := range value {
				valNum, err := strconv.ParseInt(val, 10, 0)
				if err != nil {
					return &ErrParamType{param}
				}
				valNums[index] = int(valNum)
			}
			a.nums[param.name] = valNums
			return nil

		} else {
			// must be array of string
			a.strs[param.name] = value
			return nil
		}
	}
}

// constructor
func makeApiParamList() *apiParamList {
	return &apiParamList{
		num: make(map[string]int),
		nums: make(map[string][]int),
		str: make(map[string]string),
		strs: make(map[string][]string),
	}
}

