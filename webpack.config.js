const path = require('path');
const MiniCssExtractPlugin = require("mini-css-extract-plugin");

module.exports = {
    entry: {
        index: './src/js/index.jsx'
    },
    mode: 'development',
    output: {
        path: path.resolve(__dirname, 'public/dist'),
        filename: '[name].js'
    },
    module: {
        rules: [
            {
                test: /\.jsx$/,
                use: [
                    {
                        loader: 'babel-loader',
                        options: {
                          presets: ['@babel/preset-env', '@babel/preset-react']
                        }
                    },
                ]
            },
            {
                test: /\.css$/,
                use: [
                    MiniCssExtractPlugin.loader,
                    "css-loader",
                ]
            },
            {
                test: /\.(png|jpe|jpeg|gif|woff|woff2|eot|ttf|svg)(\?.*$|$)/,
                use: {
                    loader: 'file-loader',
                },
            },
        ]
    },
    plugins: [
        new MiniCssExtractPlugin({
          filename: '[name].css'
        })
    ],
};