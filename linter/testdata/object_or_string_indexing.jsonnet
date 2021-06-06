local foo = if true then {"foo": "bar"} else "foo";
[
    foo[0],
    foo["foo"]
]
