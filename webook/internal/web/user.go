package web

import (
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/service"
	"time"
)

const (
	// 首字母小写 ，所以只有同 package web 下才可以访问该变量
	// 因为字符串中含有反斜杠\ 所以用反引号会更清爽
	emailRegexPattern = `^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$`
	// 至少包含1个字母、1个数字、1个特殊字符且密码总长度至少为8个字符
	passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`

	bizLogin = "login"
)

type UserHandler struct {
	jwtHandler

	emailRexExp    *regexp.Regexp
	passwordRexExp *regexp.Regexp

	svc     service.UserService
	codeSvc service.CodeService
}

func NewUserHandler(svc service.UserService, codeSvc service.CodeService) *UserHandler {
	return &UserHandler{
		// note 预编译正则表达式来提高校验速度
		emailRexExp:    regexp.MustCompile(emailRegexPattern, regexp.None),
		passwordRexExp: regexp.MustCompile(passwordRegexPattern, regexp.None),

		svc:     svc,
		codeSvc: codeSvc,
	}
}

func (h *UserHandler) RegisterRoutes(server *gin.Engine) {
	// note 分组路由
	g := server.Group("/users")
	// note 第二个参数本质上是 ...HandlerFunc （参数为*gin.Context的匿名函数）
	g.POST("/signup", h.SignUp)
	//g.POST("/login", h.Login)
	g.POST("/login", h.LoginJWT)
	g.GET("/profile", h.profile)
	g.POST("/edit", h.edit)
	g.POST("/login_sms/code/send", h.SendSMSLoginCode)
	g.POST("/login_sms", h.LoginSMS)
}

func (h *UserHandler) SignUp(ctx *gin.Context) {
	type Req struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		// note Bind出错，返回的是400
		ctx.JSON(http.StatusBadRequest, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	// 1. 校验两次密码是否一致
	if req.Password != req.ConfirmPassword {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "两次密码不一致",
		})
		return
	}
	// 2. 校验邮箱和密码的正则
	isEmail, err := h.emailRexExp.MatchString(req.Email)
	if err != nil {
		// 如果正则表达式本身没有错的话，是不会进入这个分支的（除非别的同事篡改）
		// note 所以单元测试，测不了这个err分支，只能测到下面 if !isEmail 这个分支
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	if !isEmail {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "邮箱格式不正确",
		})
		return
	}
	isPassword, err := h.passwordRexExp.MatchString(req.Password)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	if !isPassword {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "密码格式不正确，至少包含1个字母、1个数字、1个特殊字符且密码总长度至少为8个字符",
		})
		return
	}
	err = h.svc.SignUp(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	})
	switch err {
	case nil:
		ctx.JSON(http.StatusOK, Result{
			Msg: "注册成功",
		})
	case service.ErrDuplicateUser:
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "邮箱冲突",
		})
	default:
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
}

func (h *UserHandler) LoginJWT(ctx *gin.Context) {
	type Req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	user, err := h.svc.Login(ctx, req.Email, req.Password)
	switch err {
	case nil:
		h.setJWTToken(ctx, user.Id)
		ctx.JSON(http.StatusOK, Result{
			Data: user,
			Msg:  "登录成功",
		})
	case service.ErrInvalidEmailOrPassword:
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "邮箱或密码错误",
		})
	default:
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
}

func (h *UserHandler) Login(ctx *gin.Context) {
	type Req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	user, err := h.svc.Login(ctx, req.Email, req.Password)
	switch err {
	case nil:
		// 创建一个session
		session := sessions.Default(ctx)
		// 为了在profile等接口取出uid
		session.Set("userId", user.Id)
		session.Options(sessions.Options{
			// note sessions.Options可以理解为初始化存ssid的Cookie【在login的响应Header中的setCookie中会有展示max-age】
			// note 但 MaxAge 这一属性不同，在redis实现中，同时控制了cookie的过期时间，和session数据的过期时间（userId）
			MaxAge: 30,
		})
		err := session.Save()
		if err != nil {
			// 不太可能进入该分支
			panic("session.Save()报错")
		}

		ctx.JSON(http.StatusOK, Result{
			Data: user,
			Msg:  "登录成功",
		})

	case service.ErrInvalidEmailOrPassword:
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "邮箱或密码错误",
		})
	default:
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
}

func (h *UserHandler) profile(ctx *gin.Context) {

	/*
		// 方式一：从session中取uid
		session := sessions.Default(ctx)
		userId := session.Get("userId").(int64)
	*/

	// 方式二：从jwt中取uid
	// note ctx.get("user")还得判断ok才能类型断言，所以用MustGet()
	uc := ctx.MustGet("user").(UserClaims)
	userId := uc.Uid

	u, err := h.svc.FindById(ctx, userId)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		return
	}
	// note 不能直接将domain返回给前端，重新构建一个struct，并返回指定字段
	type User struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Birthday string `json:"birthday"`
		Resume   string `json:"resume"`
	}

	ctx.JSON(http.StatusOK, User{
		Nickname: u.Nickname,
		Email:    u.Email,
		Birthday: u.Birthday.Format(time.DateOnly),
		Resume:   u.Resume,
	})

}

func (h *UserHandler) edit(ctx *gin.Context) {
	/*
		// 方式一：从session中取uid
		session := sessions.Default(ctx)
		userId := session.Get("userId").(int64)
	*/

	// 方式二：从jwt中取uid
	// note ctx.get("user")还得判断ok才能类型断言，所以用MustGet()
	uc := ctx.MustGet("user").(UserClaims)
	userId := uc.Uid

	type Req struct {
		Nickname string `json:"nickname"`
		Birthday string `json:"birthday"`
		Resume   string `json:"resume"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		return
	}

	// note 对Birthday的校验，可以不用regex，但返回的是time类型 （忽略对 nickname 和 resume 的校验）
	birthday, err := time.Parse(time.DateOnly, req.Birthday)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "生日格式不对",
		})
		return
	}

	err = h.svc.UpdateNonSensitiveInfo(ctx, domain.User{
		Id:       userId,
		Nickname: req.Nickname,
		Birthday: birthday,
		Resume:   req.Resume,
	})

	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "修改成功",
	})
}

func (h *UserHandler) SendSMSLoginCode(ctx *gin.Context) {
	type Req struct {
		Phone string `json:"phone"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}

	// 校验一下phone
	if req.Phone == "" {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "手机号不能为空",
		})
	}

	// 调用codeSvc
	err := h.codeSvc.Send(ctx, bizLogin, req.Phone)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
	switch err {
	case nil:
		ctx.JSON(http.StatusOK, Result{
			Msg: "发送成功",
		})
	case service.ErrCodeSendTooMany:
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "发送太频繁",
		})
	default:
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}

}

func (h *UserHandler) LoginSMS(ctx *gin.Context) {
	type Req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}

	ok, err := h.codeSvc.Verify(ctx, bizLogin, req.Phone, req.Code)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
	if !ok {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "验证码错误",
		})
	}

	// 验证码正确 ==》 实现用phone注册并登录
	u, err := h.svc.FindOrCreate(ctx, req.Phone)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误",
		})
	}
	// 登录成功后，设置JWT
	h.setJWTToken(ctx, u.Id)
	ctx.JSON(http.StatusOK, Result{
		Msg: "登录成功",
	})

}
