// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

const path = require('path');
const CompressionWebpackPlugin = require('compression-webpack-plugin');
const productionGzipExtensions = ['js', 'css', 'ttf'];

module.exports = {
    publicPath: "/static/dist",
    productionSourceMap: false,
    parallel: true,
    configureWebpack: {
        plugins: [
            new CompressionWebpackPlugin({
                algorithm: 'brotliCompress',
                filename: '[path].br[query]',
                test: new RegExp('\\.(' + productionGzipExtensions.join('|') + ')$'),
                threshold: 1024,
                minRatio: 0.8
            }),
            new CompressionWebpackPlugin({
                algorithm: 'gzip',
                filename: '[path].gz[query]',
                test: new RegExp('\\.(' + productionGzipExtensions.join('|') + ')$'),
                threshold: 1024,
                minRatio: 0.8
            }),
        ],
    },
    chainWebpack: config => {
        config.output.chunkFilename(`js/vendors_[hash].js`);
        config.output.filename(`js/app_[hash].js`);

        config.resolve.alias
            .set('@', path.resolve('src'));

        config
            .plugin('html')
            .tap(args => {
                args[0].template = './index.html';
                return args
            });
    }
};
