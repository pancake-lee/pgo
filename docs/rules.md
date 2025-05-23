# 规范

## GIT

- 提交信息
  - build: 构建相关脚本
  - chore: 杂项
  - ci: ci工具更改
  - docs: 仅仅修改了文档，比如 README, CHANGELOG, CONTRIBUTE 等等
  - feat: 新增功能
  - fix: 修复 bug
  - perf: 性能优化
  - refactor: 重构（即不是新增功能，也不是修改 bug 的代码变动）
  - revert: 回滚到上一个版本
  - style: 不影响代码运行的变动
    - 如删除空格、格式化、缺失分号等等
    - 也包括：整理，文件位置，代码位置，注释内容等
  - test: 增加测试，包括单元测试、集成测试等

- 常见情况
  - 提交内容涉及多个类型的，尽可能切分为多次提交，每次提交单一类型的、单一功能的、阶段性完整的、通过编译的内容
  - 提交内容涉及多个类型的，且无法很好地切分为两次提交的，应选择最主要的类型的描述作为提交信息
    - 比如：新增一个功能，同时该功能的开发顺带修复了一个 bug ，则以 feat 开头

## 命名

- 函数
  - 函数一般表达“做什么”，即使 isVaild 也可以理解为“检查有效性”，所以函数基本可以稳定用“动宾”结构，我们确定几个常用动词，除非表达特殊操作，否则能用尽量用。这几个词汇阅读起来一眼就能认出来，可读性很强
    - C: add 增加，同时我们应该尽量让一种数据的创建入口尽可能少
    - D: del 删除，同上
    - U: edit 修改，更倾向于“主动”修改数据，如修改用户所在城市
    - U: update 更新，更倾向于“被动”更新数据，如修改用户所在城市后，更新城市用户数的统计值。
    - R: get 查询，一般情况查询接口会比较丰富，所以会搭配一些其他词汇
    - 关联关系：addTagToUser/delTagFromUser 虽然还有 bind 这样的词汇也很清晰，但是本项目就用 add...to / del...from 足够了
- 变量

  - userList 而不是 users ，因为我们会遇到 boxes enemies 等
  - userMap 如果想表达更加清晰，可以 keyToValueMap，比如 idToUserMap

- http method
  - GET（SELECT）：从服务器取出资源（一项或多项）。
  - POST（CREATE）：在服务器新建一个资源。
  - PUT（UPDATE）：在服务器更新资源（客户端提供完整资源数据）。
  - PATCH（UPDATE）：在服务器更新资源（客户端提供需要修改的资源数据）。
  - DELETE（DELETE）：从服务器删除资源。
