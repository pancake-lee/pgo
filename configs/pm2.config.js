module.exports = {
  apps : [{
      name: 'userService',
      script: 'chmod +x /backend/userService && /backend/userService',
      max_restarts: 5, 
      // watch: false, // 关闭全局监视，这会监视整个目录的文件
      watch: ["/backend/userService"],  // 只监视这一个可执行文件
      // 关键配置：使用轮询，否则容器内inotify无法检查到宿主机文件变化
      watch_options: {
        usePolling: true, // 或使用 legacyWatch: true
        interval: 5000    // 轮询间隔（毫秒）
      },
      // 变更后延迟2秒再重启，等待上传/更新完成，否则文件开始写时会触发多次重启
      watch_delay: 2000,
    }
  ]
};
