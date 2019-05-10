{
    bar: "right",
} + {
    local foo = super.bar,
    bar: "wrong1",
    answer: foo,
} + {
    bar: "wrong2"
}