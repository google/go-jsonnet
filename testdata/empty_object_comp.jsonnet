[
    {} + { [x]: 42 for x in [] },
    { [x]: 42 for x in [] } + {},
    std.objectFields({ [x]: 42 for x in [] })
]
