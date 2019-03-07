local g() = true;
local f(x) = g() && x;

f(true) tailstrict
