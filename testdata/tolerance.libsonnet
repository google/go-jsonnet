{
    // Formats a floating point number to get 15 digits
    // after the decimal point. It is used for some tests
    // of numeric functions, where a range of values is 
    // acceptable, due to differences in cpu architecture.
    tolerance(x):: "%.15f" % x
}
