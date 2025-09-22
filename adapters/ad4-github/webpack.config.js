const path = require('path');

module.exports = {
  entry: './src/main.js',
  target: 'node',
  mode: 'production',
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: 'index.js',
    libraryTarget: 'commonjs2'
  },
  externals: {
    '@actions/core': 'commonjs @actions/core',
    '@actions/github': 'commonjs @actions/github',
    '@actions/artifact': 'commonjs @actions/artifact'
  },
  optimization: {
    minimize: true,
    usedExports: true
  },
  resolve: {
    extensions: ['.js', '.json']
  },
  module: {
    rules: [
      {
        test: /\.js$/,
        exclude: /node_modules/,
        use: {
          loader: 'babel-loader',
          options: {
            presets: ['@babel/preset-env']
          }
        }
      }
    ]
  },
  node: {
    __dirname: false,
    __filename: false
  }
};
