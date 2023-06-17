// This test is intended to check that the whole stdlib is present
// and that all function have the right parameter names

// Functions without optional arguments need only one line
// Functions with optional arguments need two lines - one with none of the optional arguments
// and the other with all of them.

{
    // extVar and native are skipped here, because of the special setup required.
    // We also skip undocumented functions used in desugaring and std.trace.
    thisFile: std.thisFile,

    // Types and reflection
    type: std.type(x={}),
    length: std.length(x=[]),
    objectHas: std.objectHas(o={}, f="fieldname"),
    objectFields: std.objectFields(o={}),
    objectValues: std.objectValues(o={}),
    objectKeysValues: std.objectKeysValues(o={}),
    objectHasAll: std.objectHasAll(o={}, f="fieldname"),
    objectFieldsAll: std.objectFieldsAll(o={}),
    objectValuesAll: std.objectValuesAll(o={}),
    objectKeysValuesAll: std.objectKeysValuesAll(o={}),
    prune: std.prune(a={x: null, y: [null, "42"]}),
    mapWithKey: std.mapWithKey(func=function(key, value) 42, obj={a: 17}),
    get: [
        std.get(o={a:: 17}, f="a"),
        std.get(o={a:: 17}, f="a", default=42, inc_hidden=false),
    ],

    // isSomething
    isArray: std.isArray(v=[]),
    isBoolean: std.isBoolean(v=true),
    isFunction: std.isFunction(v=function() 42),
    isNumber: std.isNumber(v=42),
    isObject: std.isObject(v={}),
    isString: std.isString(v=""),

    // Mathematical utilities
    abs: std.abs(n=-42),
    sign: std.sign(n=17),
    max: std.max(a=2, b=3),
    min: std.min(a=2, b=3),
    pow: std.pow(x=2, n=3),
    exp: std.exp(x=5),
    log: std.log(x=5),
    exponent: std.exponent(x=5),
    mantissa: std.mantissa(x=5),
    floor: std.floor(x=5),
    ceil: std.ceil(x=5),
    sqrt: std.sqrt(x=5),
    sin: std.sin(x=5),
    cos: std.cos(x=5),
    tan: std.tan(x=5),
    asin: std.asin(x=0.5),
    acos: std.acos(x=0.5),
    atan: std.atan(x=5),

    // Assertions and debugging
    assertEqual: std.assertEqual(a="a", b="a"),

    // String Manipulation
    toString: std.toString(a=42),
    codepoint: std.codepoint(str="A"),
    char: std.char(n=65),
    substr: std.substr(str="test", from=2, len=1),
    findSubstr: std.findSubstr(pat="test", str="test test"),
    startsWith: std.startsWith(a="jsonnet", b="json"),
    endsWith: std.endsWith(a="jsonnet", b="sonnet"),
    stripChars: std.stripChars(str="aaabbbbcccc", chars="ac"),
    lstripChars: std.lstripChars(str="aaabbbbcccc", chars="a"),
    rstripChars: std.rstripChars(str="aaabbbbcccc", chars="c"),
    split: std.split(str="a,b,c", c=","),
    splitLimit: std.splitLimit(str="a,b,c", c=",", maxsplits=1),
    strReplace: std.strReplace(str="aaa", from="aa", to="bb"),
    asciiUpper: std.asciiUpper(str="Blah"),
    asciiLower: std.asciiLower(str="Blah"),
    stringChars: std.stringChars(str="blah"),
    format: std.format(str="test %s %d", vals=["blah", 42]),

    // TODO(sbarzowski) fix mismatch in the parameter name between docs and the implementations.
    escapeStringBash: std.escapeStringBash(str_="test \'test\'test"),
    escapeStringDollars: std.escapeStringDollars(str_="test \'test\'test"),
    escapeStringJson: std.escapeStringJson(str_="test \'test\'test"),

    escapeStringPython: std.escapeStringPython(str="test \'test\'test"),

    // Parsing

    parseInt: std.parseInt(str="42"),
    parseOctal: std.parseOctal(str="123"),
    parseHex: std.parseHex(str="DEADBEEF"),
    parseJson: std.parseJson(str='{"a": "b"}'),
    encodeUTF8: std.encodeUTF8(str="blah"),
    decodeUTF8: std.decodeUTF8(arr=[65, 65, 65]),

    // Manifestation

    manifestIni: std.manifestIni(ini={main: {a: 1, b:2}, sections: {s1: {x: 1, y: 2}}}),
    manifestPython: std.manifestPython(v={a: {b: "c"}}),
    manifestPythonVars: std.manifestPythonVars(conf={a: {b: "c"}}),
    manifestTomlEx: std.manifestTomlEx(value={a: {b: "c"}}, indent=" "),
    manifestJsonEx: std.manifestJsonEx(value={a: {b: "c"}}, indent=" "),
    manifestJsonMinified: std.manifestJsonMinified(value={a: {b: "c"}}),
    manifestYamlDoc: std.manifestYamlDoc(value={a: {b: "c"}}),
    manifestYamlStream: std.manifestYamlStream(value=[42, {a: {b: "c"}}]),
    manifestXmlJsonml:  std.manifestXmlJsonml(value=["blah", {a: 42}]),


    // Arrays

    makeArray: std.makeArray(sz=5, func=function(i) i),
    count: std.count(arr=["a", "b", "c", "c"], x="b"),
    find: std.find(value=42, arr=[1, 2, 42, 3, 42]),
    member: std.member(arr=[1, 2, 3], x=2),
    map: std.map(func=function(x) -x, arr=[1, 2, 3]),
    mapWithIndex: std.mapWithIndex(func=function(i, x) i + x, arr=[3, 2, 1]),
    filterMap: std.filterMap(filter_func=function(x) x % 2 == 0, map_func=function(x) x * 2, arr=[1, 2, 3, 4]),
    flatMap: std.flatMap(func=function(x) [x*2, x*3], arr=[1,2,3]),
    filter: std.filter(func=function(x) x % 2 == 0, arr=[1, 2, 3, 4]),
    foldl: std.foldl(func=function(x, y) x + y, arr=[[1], [2], [3]], init=[0]),
    foldr: std.foldr(func=function(x, y) x + y, arr=[[1], [2], [3]], init=[4]),
    repeat: std.repeat(what="foo", count=3),
    slice: std.slice(indexable="foobar", index=1, end=2, step=1),
    range: std.range(from=1, to=5),
    join:  std.join(sep=",", arr=["a", "b", "c"]),
    lines: std.lines(arr=["a", "b", "c"]),
    flattenArrays: std.flattenArrays([[1], [2, 3], [4, 5, [6, 7]]]),
    reverse: std.reverse(["b", "a"]),
    sort: [
        std.sort([2, 3, 1]),
        std.sort(arr=[2, 3, 1], keyF=function(x) -x),
    ],
    uniq: [
        std.uniq([1, 2, 2, 3]),
        std.uniq(arr=["a", "B", "b", "a"], keyF=std.asciiLower),
    ],

    // Sets

    set: [
        std.set([2, 3, 1]),
        std.set(arr=[2, 3, 1], keyF=function(x) -x),
    ],
    setInter: [
        std.setInter([1,2,3], [3,4,5]),
        std.setInter(a=[1,2,3], b=[3,4,5], keyF=function(x) std.floor(x / 2)),
    ],
    setUnion: [
        std.setUnion([1,2,3], [3,4,5]),
        std.setUnion(a=[1,2,3], b=[3,4,5], keyF=function(x) std.floor(x / 2)),
    ],
    setDiff: [
        std.setDiff([1,2,3], [3,4,5]),
        std.setDiff(a=[1,2,3], b=[3,4,5], keyF=function(x) std.floor(x / 2)),
    ],
    setMember: [
        std.setMember(3, [2]),
        std.setMember(x=3, arr=[2], keyF=function(x) std.floor(x / 2)),
    ],

   // Encoding

    base64: [
        std.base64(input=std.encodeUTF8("blah")),
        std.base64(input="blah"),
    ],
    base64DecodeBytes: std.base64DecodeBytes(str="YmxhaAo="),
    base64Decode: std.base64Decode(str="YmxhaAo="),
    md5: std.md5(s="md5"),

    // JSON Merge Patch

    "mergePatch": std.mergePatch(target={a: 42}, patch={ a: null }),

}
