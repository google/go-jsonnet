{
    local a = 42,
    local b = {
        local c = a,
        f: c
    },
    f: b
}