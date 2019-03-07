local g() = false;
local f(x) = g() || x;

f(true) tailstrict
