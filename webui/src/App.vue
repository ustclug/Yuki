<style scoped>
.layout{
  border: 1px solid #d7dde4;
  background: #f5f7f9;
  position: relative;
  border-radius: 4px;
  overflow: hidden;
}
.layout-content{
  min-height: 200px;
  margin: 15px;
  overflow: hidden;
  background: #fff;
  border-radius: 4px;
}
.layout-content-main{
  padding: 10px;
}
.layout-copy{
  text-align: center;
  padding: 10px 0 20px;
  color: #9ea7b4;
}
.layout-menu-left{
  background: #464c5b;
}
.layout-header{
  height: 60px;
  background: #fff;
  box-shadow: 0 1px 1px rgba(0,0,0,.1);
}
.layout-logo-left{
  width: 90%;
  height: 30px;
  background: #5b6270;
  border-radius: 3px;
  margin: 15px auto;
}
.layout-ceiling-main a{
  color: #9ba7b5;
}
.layout-hide-text .layout-text{
  display: none;
}
.ivu-col{
  transition: width .2s ease-in-out;
}
#repo-table {
}
</style>
<template>
  <div id="app" class="layout" :class="{'layout-hide-text': false }">
    <Row type="flex">
    <i-col :span="4" class="layout-menu-left">
      <Menu active-name="1" theme="dark" width="auto">
        <div class="layout-logo-left"></div>
        <Menu-item name="1">
          <Icon type="ios-navigate" :size="24"></Icon>
          <span class="layout-text">Repository Status</span>
      </Menu-item>
      </Menu>
      </i-col>
      <i-col :span="20">
        <div class="layout-header">
          <i-button type="text" @click="toggleClick">
            <Icon type="navicon" size="32"></Icon>
            </i-button>
        </div>
        <div class="layout-content">
          <div class="layout-content-main">
            <Table id="repo-table" border :columns="repoTable" :data="repos"></Table>
          </div>
        </div>
        <div class="layout-copy">
          USTC Mirror
        </div>
        </i-col>
    </Row>
  </div>
</template>

<script>
import moment from 'moment'

const toPrettySize = (size) => {
  const units = ['B', 'KB', 'MB', 'GB']
  for (const unit of units) {
    if (size < 1024) {
      return `${size.toFixed(2)} ${unit}`
    } else {
      size /= 1024
    }
  }
  return `${size.toFixed(2)} TB`
}

const toLocalTime = (time) => moment(time).local().format('YYYY-MM-DD HH:mm:ss')

export default {
  name: 'app',
  data () {
    return {
      repos: [],
      repoTable: [{
        title: 'Archive Name',
        sortable: true,
        align: 'center',
        key: '_id'
      }, {
        title: 'Upstream',
        align: 'center',
        key: 'upstream'
      }, {
        title: 'Last Success',
        align: 'center',
        sortable: true,
        key: 'lastSuccess'
      }, {
        title: 'Last Status',
        align: 'center',
        render(row, col, idx) {
          return row.lastExitCode === 0 ? 'success' : 'failure'
        },
        filters: [{
          label: 'Success',
          value: 0,
        }, {
          label: 'Failure',
          value: 1,
        }],
        filterMethod(val, row) {
          if (val === 0) return row.lastExitCode === 0
          else return row.lastExitCode !== 0
        },
        key: 'lastExitCode'
      }, {
        title: 'Updated At',
        align: 'center',
        sortable: true,
        key: 'updatedAt'
      }, {
        title: 'Size',
        align: 'center',
        sortable: true,
        key: 'size',
        render(row, col, idx) {
          return toPrettySize(row.size)
        }
      }]
    }
  },
  methods: {
    fetchData(url) {
      this.$http
        .get(url)
        .then((resp) => {
          this.repos = resp.data.map(r => {
            r.updatedAt = toLocalTime(r.updatedAt)
            r.lastSuccess = toLocalTime(r.lastSuccess)
            return r
          })
        })
    },
  },
  created() {
    this.fetchData('/api/v1/meta')
  },
}
</script>
