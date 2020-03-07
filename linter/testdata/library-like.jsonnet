local lib = {
    number: import "42.jsonnet",
    addTwoNumbers(x, y): x + y,
    addTwoNumbers2: lib.addTwoNumbers,
    addTwoNumbers3: self.addTwoNumbers,
    nested: {
        twoObjects(): [{a: "a"}, {b: "b"}],
    }
}; lib
