local cmpTest(a, b) = {
    a: a,
    b: b,
    '<': a < b,
    '>': a > b,
    '<=': a <= b,
    '>=': a >= b,
};
[
    cmpTest(1, 2),
    cmpTest(2, 1),
    cmpTest([1], [2]),
    cmpTest([2], [1]),
    cmpTest([], []),
    cmpTest([], [1]),
    cmpTest([1], []),
    cmpTest([1, 2], [1]),
    cmpTest([1, 2], [2]),
    cmpTest([[1]], [[2]]),
    cmpTest([[2]], [[1]]),
    cmpTest(["foo"], ["bar"]),
    cmpTest([0, "a"], [0, "b"]),
    cmpTest("foo", "bar"),
]
