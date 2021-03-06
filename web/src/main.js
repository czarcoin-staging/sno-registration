// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

import Vue from 'vue';
import VueClipboard from 'vue-clipboard2';
import VueLazyload from 'vue-lazyload';

import App from './App.vue';

Vue.config.productionTip = false;

Vue.use(VueClipboard);

Vue.use(VueLazyload, {
    lazyComponent: true,
});

new Vue({
    render: (h) => h(App),
}).$mount('#app');
