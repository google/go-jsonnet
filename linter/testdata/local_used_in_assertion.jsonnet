{
    local input = [1,2,3],
    local knownItems = [1,2],
    local unknownItems = std.filter(function(i) !(i in knownItems), input),
    assert unknownItems == [] : "unexpected items: %s" % std.join(",",unknownItems)
}