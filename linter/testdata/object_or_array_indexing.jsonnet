local foo = if true then {"foo": "bar"} else ["f", "o", "o"];
[
    foo[0],
    foo["foo"]
]
