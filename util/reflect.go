package util

import "reflect"

//set the struct into map, the key of map is the json tag or
//field name if the tag not exist
func Struct2Map(obj interface{}) map[string]interface{} {
	v := reflect.ValueOf(obj)
	t := v.Elem()
	typeOfType := t.Type()

	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		var f string
		if typeOfType.Field(i).Tag.Get("json") == "" {
			f = typeOfType.Field(i).Name
		} else {
			f = typeOfType.Field(i).Tag.Get("json")
		}
		data[f] = t.Field(i).Interface()
	}
	return data
}
