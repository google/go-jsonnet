local foo = { config: [{ x: 'y' }] };
foo {
  config: [super.config[0] { a: 'b' }],
}
