package main

import "encoding/json"

func PrettyPrintJson(i interface{}) string {
    s, _ := json.MarshalIndent(i, "", "  ")
    return string(s)
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}
