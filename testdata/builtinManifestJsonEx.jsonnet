local a = {
    foo: "bar",
    bar: self.foo,
    baz: 1,
    bazel: 1.42,
    boom: -1,
    bim: false,
    bam: true,
    blamo: {
        cereal: [
            "<>& fizbuzz",
        ],

        treats: [
            {
                name: "chocolate",
            }
        ],
    }
};

std.manifestJsonEx(a, "  ")