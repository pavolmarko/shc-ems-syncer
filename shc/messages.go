package shc

type registerMsg struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	PrimaryRole string `json:"primaryRole"`
	Certificate string `json:"certificate"`
}
