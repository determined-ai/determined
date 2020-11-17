module.exports = {
  mode: 'development',
  module: {
    rules: [
      {
        exclude: [ /node_modules/ ],
        // every time webpack sees a TS file (except for node_modules)
        // webpack will use "ts-loader" to transpile it to JavaScript
        test: /\.ts$/,
        use: [
          {
            loader: 'ts-loader',
            options: {
              // skip typechecking for speed
              transpileOnly: true,
            },
          },
        ],
      },
    ],
  },
  // webpack will transpile TS and JS files
  resolve: {
    extensions: [ '.ts', '.js' ],
  },
};
