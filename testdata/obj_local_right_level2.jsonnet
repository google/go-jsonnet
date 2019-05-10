{
    local foo = self.bar,
    bar: "wrong",
    answer: foo
} + {
    bar: "right"
}