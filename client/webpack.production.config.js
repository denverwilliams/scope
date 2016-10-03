var webpack = require('webpack');
var autoprefixer = require('autoprefixer');
var path = require('path');

var CleanWebpackPlugin = require('clean-webpack-plugin');
var ExtractTextPlugin = require("extract-text-webpack-plugin");
var HtmlWebpackPlugin = require('html-webpack-plugin');

var GLOBALS = {
  'process.env': {NODE_ENV: '"production"'}
};

/**
 * This is the Webpack configuration file for production.
 */
module.exports = {

  // fail on first error when building release
  bail: true,

  cache: {},

  entry: {
    app: './app/scripts/main',
    'contrast-app': './app/scripts/contrast-main',
    'terminal-app': './app/scripts/terminal-main',
    // keep only some in here, to make vendors and app bundles roughly same size
    vendors: ['babel-polyfill', 'classnames', 'd3', 'immutable',
      'lodash', 'react', 'react-dom', 'react-redux',
      'redux', 'redux-thunk']
  },

  output: {
    path: path.join(__dirname, 'build/'),
    filename: '[chunkhash].js',
    // allow a custom public path to be passed in as part of the build process,
    // this is useful if you want to serve static content from a CDN, etc.
    publicPath: process.env.STATIC_CONTENT_PATH || undefined
  },

  module: {
    include: [
      path.resolve(__dirname, 'app/scripts')
    ],
    preLoaders: [
      {
        test: /\.js$/,
        exclude: /node_modules|vendor/,
        loader: 'eslint-loader'
      }
    ],
    loaders: [
      {
        test: /\.less$/,
        loader: ExtractTextPlugin.extract('style-loader',
          'css-loader?minimize!postcss-loader!less-loader')
      },
      {
        test: /\.woff(2)?(\?v=[0-9]\.[0-9]\.[0-9])?$/,
        loader: 'url-loader?limit=10000&minetype=application/font-woff'
      },
      {
        test: /\.(ttf|eot|svg|ico)(\?v=[0-9]\.[0-9]\.[0-9])?$/,
        loader: 'file-loader'
      },
      {
        test: /\.ico$/,
        loader: 'file-loader?name=[name].[ext]'
      },
      { test: /\.jsx?$/, exclude: /node_modules|vendor/, loader: 'babel' }
    ]
  },

  postcss: [
    autoprefixer({
      browsers: ['last 2 versions']
    })
  ],

  eslint: {
    failOnError: true
  },

  resolve: {
    extensions: ['', '.js', '.jsx']
  },

  plugins: [
    new CleanWebpackPlugin(['build']),
    new webpack.DefinePlugin(GLOBALS),
    new webpack.optimize.CommonsChunkPlugin('vendors', '[chunkhash].js'),
    new webpack.optimize.OccurenceOrderPlugin(true),
    new webpack.IgnorePlugin(/^\.\/locale$/, [/moment$/]),
    new webpack.optimize.UglifyJsPlugin({
      sourceMap: false,
      compress: {
        warnings: false
      }
    }),
    new ExtractTextPlugin('style-[name].css'),
    new HtmlWebpackPlugin({
      hash: true,
      chunks: ['vendors', 'contrast-app'],
      template: 'app/html/index.html',
      filename: 'contrast.html'
    }),
    new HtmlWebpackPlugin({
      hash: true,
      chunks: ['vendors', 'terminal-app'],
      template: 'app/html/index.html',
      filename: 'terminal.html'
    }),
    new HtmlWebpackPlugin({
      hash: true,
      chunks: ['vendors', 'app'],
      template: 'app/html/index.html',
      filename: 'index.html'
    })
  ]
};
