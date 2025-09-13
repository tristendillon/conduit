package profile_repo

type Profile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var profiles = []Profile{
	{ID: "1", Name: "John Doe", Email: "john.doe@example.com"},
	{ID: "2", Name: "Jane Doe", Email: "jane.doe@example.com"},
}

func DeleteProfile(id string) int {
	index := FindProfileIndex(id)
	if index == -1 {
		return index
	}
	profiles = append(profiles[:index], profiles[index+1:]...)
	return index
}

func GetAllProfiles() []Profile {
	return profiles
}

func FindProfile(id string) *Profile {
	for i := range profiles {
		if profiles[i].ID == id {
			return &profiles[i]
		}
	}
	return nil
}

func FindProfileIndex(id string) int {
	for i := range profiles {
		if profiles[i].ID == id {
			return i
		}
	}
	return -1
}
