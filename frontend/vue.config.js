const { defineConfig } = require("@vue/cli-service");
module.exports = defineConfig({
  parallel: false,
  transpileDependencies: true,
  devServer: {
    proxy: {
      "/api": {
        target: "http://127.0.0.1:9095",
        changeOrigin: true, // 允许跨域
      },
    },
  },
});
