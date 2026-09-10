package main

import "encoding/json"

func jsonUnmarshalString(raw string, v any) error {
	return json.Unmarshal([]byte(raw), v)
}
