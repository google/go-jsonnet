local input = std.makeArray(10000, function(i) 'xxxxx');

std.foldl(function(acc, value) acc + value, input, '')
