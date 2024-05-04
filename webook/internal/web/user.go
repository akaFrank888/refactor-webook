package web

import (
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
)

type UserHandler struct {
	emailRexExp    *regexp.Regexp
	passwordRexExp *regexp.Regexp

	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{
		// note 预编译正则表达式来提高校验速度
		emailRexExp:    regexp.MustCompile(emailRegexPattern, regexp.None),
		passwordRexExp: regexp.MustCompile(passwordRegexPattern, regexp.None),

		svc: svc,
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
}

func (h *UserHandler) SignUp(ctx *gin.Context) {
	type Req struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	var req Req
	if err := ctx.Bind(&req); err != nil {
		ctx.JSON(http.StatusOK, Result{
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
	case service.ErrUserDuplicateEmail:
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
		uc := UserClaims{
			Uid: user.Id,
			// 定义JWT过期时间 —— 1min
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			},
		}
		// 此token只是jwt的一个token结构体
		token := jwt.NewWithClaims(jwt.SigningMethodHS512, uc)
		// 此tokenStr才是传输的token

		tokenStr, err := token.SignedString(JWTKey)
		if err != nil {
			ctx.JSON(http.StatusOK, Result{
				Code: 5,
				Msg:  "系统错误",
			})
		}
		// 添加进响应的header中
		ctx.Header("x-jwt-token", tokenStr)
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
	// note ctx.get("user")还得判断ok才能类型断言，所以用MustGet()
	// uc := ctx.MustGet("user").(UserClaims)

}

var JWTKey = []byte("oIft1b5qZjyLcc0zZo2UrUx5rk3KE0LvZKv73fw502oXd6vfYu1OAQvbSel8whvm")

type UserClaims struct {
	jwt.RegisteredClaims
	Uid int64
}
