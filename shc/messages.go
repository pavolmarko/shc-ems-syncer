package shc

type registerMsg struct {
	Type        string `json:"@type"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	PrimaryRole string `json:"primaryRole"`
	Certificate string `json:"certificate"`
}
