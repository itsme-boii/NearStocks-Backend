package cutils

func Listify[T interface{}](slicePtr *[]T) *[]T {
	if slicePtr == nil || slicePtr == &[]T{} {
		return &[]T{}
	}
	return slicePtr
}
