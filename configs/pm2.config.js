module.exports = {
  apps : [{
      name: 'userService',
      script: 'chmod +x /backend/userService && /backend/userService',
      watch: true,
      max_restarts: 5,
    }
  ]
};
