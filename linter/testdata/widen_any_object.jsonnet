local unknown = (function(x) x)({});
if true then
    unknown
else {
    foo: "bar"
}
