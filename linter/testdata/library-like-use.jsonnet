local lib = import "library-like.jsonnet";
[
    lib.number,
    lib.number(),
    lib.addTwoNumbers(1, 2),
    lib.addTwoNumbers(1, 2, 3),
    lib.addTwoNumbers("a", "b"),
    lib.addTwoNumbers(1, 2)[2],
    lib.addTwoNumbers2(1, 2)[2],
    lib.addTwoNumbers2(1, 2, 3),
    lib.addTwoNumbers3(1, 2)[2],
    lib.addTwoNumbers3(1, 2, 3),
    lib.nested.twoObjects[0].a,
    lib.nested.twoObjects[0].b,
    lib.nested.twoObjects[1].a,
    lib.nested.twoObjects[1].b,
    lib.nested.nonexistent,
    lib.nonexistent,
]
