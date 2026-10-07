local outer = 7;
local parameterDefault(a, b=a) = a + b;
local outerDefault(a=outer) = a;
local chainedDefaults(a=1, b=a+1, c=b+1) = c;
local namedDefault(a, b=a) = b;
local objectDefault = { value: 5, f(a=self.value): a };
local inheritedDefault = { value: 3 } + { f(a=super.value): a };
local recursiveDefault(n, step=1) =
  if n == 0 then n else recursiveDefault(n-step) tailstrict;
{
  parameter: parameterDefault(1) tailstrict,
  outer: outerDefault() tailstrict,
  chain: chainedDefaults() tailstrict,
  named: namedDefault(a=2) tailstrict,
  explicit: parameterDefault(1, 1) tailstrict,
  lazy: parameterDefault(1),
  recursive: recursiveDefault(1000) tailstrict,
  'self': objectDefault.f() tailstrict,
  'super': inheritedDefault.f() tailstrict,
}
