package service

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository"
)

var (
	ErrDuplicateUser          = repository.ErrDuplicateUser
	ErrInvalidEmailOrPassword = repository.ErrUserNotFound
)

type UserService interface {
	SignUp(ctx context.Context, user domain.User) error
	Login(ctx context.Context, email, password string) (domain.User, error)
	FindById(ctx context.Context, uid int64) (domain.User, error)
	UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error
	FindOrCreate(ctx context.Context, phone string) (domain.User, error)
}

// 避免与接口同名，所以小写首字母
type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{
		repo: repo,
	}
}

func (svc *userService) SignUp(ctx context.Context, user domain.User) error {
	// 若认为加密是业务方面的事就将bcrypt加密写在service中（若是存储方面的事就写在repo中）
	EncryptPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(EncryptPassword)
	return svc.repo.Create(ctx, user)
}

func (svc *userService) Login(ctx context.Context, email, password string) (domain.User, error) {

	user, err := svc.repo.FindByEmail(ctx, email)
	if err != nil {
		// note 注意：不管是用户没找到，还是密码错误，都返回同一个err
		return domain.User{}, ErrInvalidEmailOrPassword
	}
	// 核对密码 参数分别是：加密后的密码 和 输入的明文密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		// note 注意：不管是用户没找到，还是密码错误，都返回同一个err
		return domain.User{}, ErrInvalidEmailOrPassword
	}
	return user, nil
}
func (svc *userService) FindById(ctx context.Context, uid int64) (domain.User, error) {
	return svc.repo.FindById(ctx, uid)

}
func (svc *userService) UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error {
	return svc.repo.UpdateNonZeroFields(ctx, user)
}

// FindOrCreate note “查找-做某事”的场景一定有并发问题，而该用手机号登录并注册的场景中的并发问题是，可能会注册同一个phone的两个用户，且肯定是攻击者操作（因为普通用户不会的手机号仅为自己所有，不可能存在多个用户用同一个手机号填写正确的验证码后进行提交）
// note 解决方式是： 1. 设置phone为唯一索引 2. 对因并发问题发生的唯一索引冲突err对handler层屏蔽，直接返回已有user作为处理即可
func (svc *userService) FindOrCreate(ctx context.Context, phone string) (domain.User, error) {
	u, err := svc.repo.FindByPhone(ctx, phone)
	if err != repository.ErrUserNotFound {
		// 大部分情况都是老用户
		return u, err
	}
	// 小部分情况是新用户
	err = svc.repo.Create(ctx, domain.User{
		Phone: phone,
	})
	if err != nil && err != repository.ErrDuplicateUser {
		// note 仅因系统错误（排除攻击者造成的并发问题导致的err）
		return domain.User{}, err
	}
	// TODO 主从延迟 ==>插入进的是主库，查询查的是从库，所以可能刚插进去就查的话查不到，因为主从库还没同步完成【解决方式是强制查主库，但还没做】
	return svc.repo.FindByPhone(ctx, phone)
}
