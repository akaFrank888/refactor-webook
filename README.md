### week2重构与复习
（0） 学习内容：
<img src="image/week2.png">

（1）遇到的问题与解决：
1. internal包实现包的私有化：refactor-webook/webook/internal里面的代码只能被直接父包及其子包引用。所以refactor-webook/webook以外的地方调用不了internal里的内容（如refactor-webook/main.go），但refactor-webook/webook/main.go可以

（2）处理方式：
1. sign中遇到的email（unique）冲突问题是在dao层获取mysql错误码解决的。但在login中遇到的email不存在问题，可以直接在dao层声明gorm.ErrRecordNotFound的变量
2. 区分ssid与userId及其中用到的cookie和session：
   1) 在初始化web服务器时，需要初始化session，所以创建了基于cookie携带ssid的session
   2) 基于步骤1的session，才能进行登录后的session.Set和登录校验的session.Get的userId
3) sessionId可以存放在哪里？
   1) cookie
   2) header（当cookie被禁用时）
   3) 查询参数（当cookie被禁用时）