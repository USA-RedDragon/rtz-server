import mitt from 'mitt';

export default {
  install: (app, _) => {
    // inject a globally available $EventBus
    app.config.globalProperties.$EventBus = mitt();
  },
};