const config = require('@rancher/shell/pkg/vue.config');

// This vue.config is used by the build-pkg script which runs from inside this directory.
// It creates a .shell symlink here before invoking vue-cli-service.
const shellConfig = config(__dirname);

// Disable fork-ts-checker: the shell itself has TS errors we don't own,
// and ts-loader with transpileOnly:true is sufficient for building.
module.exports = {
  ...shellConfig,
  chainWebpack: (cfg) => {
    if (shellConfig.chainWebpack) shellConfig.chainWebpack(cfg);
    cfg.plugins.delete('fork-ts-checker');
  },
};
