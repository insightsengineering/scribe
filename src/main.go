package main

func main() {
    var renv_lock Renvlock
    GetRenvLock("renv.lock", &renv_lock)
    ValidateRenvLock(renv_lock)
}
