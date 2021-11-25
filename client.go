package thunderdrive

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"gopkg.in/resty.v1"
)

const BaseUrl string = "https://app.thunderdrive.io"

type Client struct {
	httpClient  *resty.Client
	userDetails UserDetails
}

func New() *Client {
	client := resty.New()
	client.SetHostURL(BaseUrl)
	// client.SetDebug(true)
	client.SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.54 Safari/537.36")

	return &Client{httpClient: client}
}

type LoginRequestPayload struct {
	Remember bool   `json:"remember"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponsePayload struct {
	Data   string `json:"data"`
	Status string `json:"status"`
}

type UserDetails struct {
	User struct {
		ID int `json:"id"`
	} `json:"user"`
}

func (c Client) getXsrfToken() string {
	targetUrl, _ := url.Parse(BaseUrl)

	var cookie *http.Cookie
	for _, cookie = range c.httpClient.GetClient().Jar.Cookies(targetUrl) {
		if cookie.Name == "XSRF-TOKEN" {
			resp, _ := url.QueryUnescape(cookie.Value)
			return resp
		}
	}

	return ""
}

func (c Client) Login(email string, password string) {
	log.Println("Logging in to Thunder Drive")
	reqPayload := LoginRequestPayload{
		Remember: true,
		Email:    email,
		Password: password,
	}

	resp, err := c.httpClient.R().
		// SetHeader().
		SetResult(&LoginResponsePayload{}).
		SetBody(reqPayload).
		Post("/secure/auth/login")

	if err != nil {
		log.Fatal(err)
	}

	log.Println(resp, err)

	response := *resp.Result().(*LoginResponsePayload)
	log.Println(c.httpClient.Cookies)

	if response.Status == "success" {
		log.Println("Logged in")
		infoDecoded, _ := base64.StdEncoding.DecodeString(response.Data)
		json.Unmarshal(infoDecoded, &c.userDetails)
	}
}

type UsageResponsePayload struct {
	Used      int64  `json:"used"`
	Available int64  `json:"available"`
	Status    string `json:"status"`
}

func (c Client) GetUsage() *UsageResponsePayload {
	log.Println("Getting space usage")
	resp, err := c.httpClient.R().
		SetResult(&UsageResponsePayload{}).
		Get("/secure/drive/user/space-usage")

	if err != nil {
		log.Fatal(err)
	}

	response := resp.Result().(*UsageResponsePayload)

	fmt.Printf("API Usage information %+v\n", response)

	return response
}

type Folder struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID int    `json:"parent_id"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Hash     string `json:"hash"`
	URL      string `json:"url"`
}

type FoldersResponsePayload struct {
	Folders []Folder `json:"folders"`
	Status  string   `json:"status"`
}

func (c Client) Folders() []Folder {
	log.Println("Getting folders")

	resp, err := c.httpClient.R().
		SetResult(&FoldersResponsePayload{}).
		Get(fmt.Sprint("/secure/drive/users/", c.userDetails.User.ID, "/folders"))

	if err != nil {
		log.Fatal(err)
	}

	response := resp.Result().(*FoldersResponsePayload)

	if response.Status == "success" {
		return response.Folders
	}

	return nil
}

type GetEntriesRequestPayload struct {
	OrderBy  string `json:"orderBy"`
	OrderDir string `json:"orderDir"`
	Page     int    `json:"page"`
}

type EntryDetails struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	FileName   string   `json:"file_name"`
	Mime       string   `json:"mime"`
	FileSize   int      `json:"file_size"`
	ParentID   int      `json:"parent_id"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
	DeletedAt  string   `json:"deleted_at"`
	Path       string   `json:"path"`
	PublicPath string   `json:"public_path"`
	Type       string   `json:"type"`
	Extension  string   `json:"extension"`
	Public     int      `json:"public"`
	Thumbnail  bool     `json:"thumbnail"`
	Hash       string   `json:"hash"`
	URL        string   `json:"url"`
	Users      []User   `json:"users"`
	Tags       []string `json:"tags"`
}

type GetEntriesResponsePayload struct {
	CurrentPage int            `json:"current_page"`
	Data        []EntryDetails `json:"data"`
	From        int            `json:"from"`
	LastPage    int            `json:"last_page"`
	NextPageURL *string        `json:"next_page_url"`
	Path        string         `json:"path"`
	PerPage     int            `json:"per_page"`
	PrevPageURL *string        `json:"prev_page_url"`
	To          int            `json:"to"`
	Total       int            `json:"total"`
}

type EntryPermissions struct {
	View     bool `json:"view"`
	Edit     bool `json:"edit"`
	Download bool `json:"download"`
}

type User struct {
	Email       string `json:"email"`
	ID          int    `json:"id"`
	Avatar      string `json:"avatar"`
	OwnsEntry   bool   `json:"owns_entry"`
	DisplayName string `json:"display_name"`
}

func (c Client) GetEntriesPage(folderId string, page int, orderBy *string, orderDir *string) GetEntriesResponsePayload {
	log.Println("Getting entries page", folderId, page)

	queryParams := map[string]string{
		"orderBy":  "updated_at",
		"orderDir": "desc",
		"page":     strconv.Itoa(page),
		"folderId": folderId,
	}

	resp, err := c.httpClient.R().
		SetResult(&GetEntriesResponsePayload{}).
		SetQueryParams(queryParams).
		Get("/secure/drive/entries")

	if err != nil {
		log.Fatal(err)
	}

	return *resp.Result().(*GetEntriesResponsePayload)
}

func (c Client) GetEntries(path string) []EntryDetails {
	log.Println("Getting entries", path)

	result := []EntryDetails{}
	var response GetEntriesResponsePayload
	page := 1
	for ok := true; ok; ok = bool(response.NextPageURL != nil) && bool(response.To != response.Total) {
		response = c.GetEntriesPage("root", page, nil, nil)
		result = append(result, response.Data...)
		page++
	}

	return result
}

func (c Client) Remove(entryIds []string) {
	log.Println("Removing", entryIds)
	_, err := c.httpClient.R().
		SetHeader("X-XSRF-TOKEN", c.getXsrfToken()).
		SetBody(map[string]interface{}{
			"_method":  "DELETE",
			"entryIds": entryIds,
		}).
		Post("/secure/drive/entries")

	if err != nil {
		log.Fatal(err)
	}
}

// TODO: Add the move function

func (c Client) Mkdir(parentId *string, name string) {
	log.Println("Creating directory", parentId, name)

	_, err := c.httpClient.R().
		SetHeader("X-XSRF-TOKEN", c.getXsrfToken()).
		SetBody(map[string]interface{}{
			"name":      name,
			"parent_id": parentId,
		}).
		Post("/secure/drive/folders")

	if err != nil {
		log.Fatal(err)
	}
}

func (c Client) Upload(parentId string, path string) {
	log.Println("Uploading file", parentId, path)

	_, err := c.httpClient.R().
		SetHeader("X-XSRF-TOKEN", c.getXsrfToken()).
		SetFile("file", path).
		SetFormData(map[string]string{
			"parent_id": parentId,
		}).
		Post("/secure/uploads")

	if err != nil {
		log.Fatal(err)
	}
}
