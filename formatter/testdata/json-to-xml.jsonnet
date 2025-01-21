local c =
  import "json2xml.libjsonnet";

local d = {
  name:

    "foo",

  children:
 ["bar", "bam"],

  attrs: {
    class1: "abc",
    numbers: [1, 2, 3, 4],
  },
};

{
  output:
  c.manifestXml(d, "elements"),
}
