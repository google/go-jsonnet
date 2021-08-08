// Make sure shadowing std does not cause problems for desugaring.
local std = {};
[
    { [x]: 17 for x in [] },
    [ x for x in [] ],
    42 % 2137,
    "foo" in {},
]