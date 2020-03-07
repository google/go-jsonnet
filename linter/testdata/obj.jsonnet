local unknown = "a" + "b";
local obj = if true then
    {
        [unknown]: 42
    }
else
    {
        foo: [1]
    }
    ;
obj.bar[0]
