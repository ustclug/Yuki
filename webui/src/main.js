import Vue from 'vue'
import axios from 'axios'
import iView from 'iview'
import 'iview/dist/styles/iview.css'
import App from './App.vue'

Vue.use(iView)
Vue.prototype.$http = axios

new Vue({
  el: '#app',
  render: h => h(App)
})
