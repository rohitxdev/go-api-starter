package auth

type AuthClient struct {
}

func (a *AuthClient) SignUp(email string, password string) error {
	return nil
}

func (a *AuthClient) LogIn(email string, password string) error {
	return nil
}

func (a *AuthClient) ChangePassword(email string, currentPassword string, newPassword string) error {
	return nil
}

func (a *AuthClient) ResetPassword(email string, token string, newPassword string) error {
	return nil
}

func (a *AuthClient) DeleteAccount(email string, password string) error {
	return nil
}
