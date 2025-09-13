package user_repo

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = []User{
	{ID: "1", Name: "John Doe", Email: "john.doe@example.com"},
	{ID: "2", Name: "Jane Doe", Email: "jane.doe@example.com"},
}

func GetAllUsers() []User {
	return users
}

func FindUserIndex(id string) int {
	for i := range users {
		if users[i].ID == id {
			return i
		}
	}
	return -1
}

func FindUser(id string) *User {
	for _, user := range users {
		if user.ID == id {
			return &user
		}
	}
	return nil
}

func DeleteUser(id string) int {
	index := FindUserIndex(id)
	if index == -1 {
		return index
	}
	users = append(users[:index], users[index+1:]...)
	return index
}
