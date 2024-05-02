package service

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"refactor-webook/webook/internal/domain"
	"refactor-webook/webook/internal/repository"
)

var (
	ErrUserDuplicateEmail     = repository.ErrUserDuplicateEmail
	ErrInvalidEmailOrPassword = repository.ErrUserNotFound
)

type UserService interface {
	SignUp(ctx context.Context, user domain.User) error
	Login(ctx context.Context, email, password string) (domain.User, error)
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
