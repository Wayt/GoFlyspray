package goflyspray

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

const (
	authenticateEndpoint string = "/index.php?do=authenticate"
	newtaskEndpoint             = "/index.php?do=newtask"
)

// Flyspray endpoint
type FSEndpoint struct {
	URL    *url.URL
	Client *http.Transport
}

// Create an endpoint object
// Be sure of your url, it ignore any format error
// Because I want it to be able to be chained with Auth function
func Endpoint(endpointURL string) *FSEndpoint {

	parsedUrl, _ := url.Parse(endpointURL)

	return &FSEndpoint{
		URL:    parsedUrl,
		Client: &http.Transport{},
	}
}

func (e *FSEndpoint) urlFor(queryPath string) string {

	queryURL, _ := url.Parse(e.URL.String())
	uri, _ := url.ParseRequestURI(queryPath)

	queryURL.Path = uri.Path
	queryURL.RawQuery = uri.RawQuery

	return queryURL.String()
}

func (e *FSEndpoint) Auth(username, password string) (*FSSession, error) {

	// Set up query url
	queryURL := e.urlFor(authenticateEndpoint)

	// Build POST form
	form := make(url.Values)
	form.Add("user_name", username)
	form.Add("password", password)
	form.Add("return_to", "/")
	form.Add("login", "Login!")
	form.Add("remember_login", "1")

	// Create the http request
	request, err := http.NewRequest("POST", queryURL, bytes.NewReader([]byte(form.Encode())))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute the request
	resp, err := e.Client.RoundTrip(request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return nil, errors.New("Failed to authenticate, bad status")
	}

	cookies := resp.Cookies()

	// Check for successful auth
	ok := false
	for _, c := range cookies {
		if c.Name == "flyspray_passhash" {
			ok = true
		}
	}

	if !ok {
		return nil, errors.New("Failed to authenticate, missing cookie")
	}

	return &FSSession{
		cookies:  cookies,
		endpoint: e,
	}, nil
}

// Flyspray session object
type FSSession struct {
	cookies  []*http.Cookie // Session cookies, used for auth
	endpoint *FSEndpoint    // Session endpoint
}

// New Task Form
type NewTaskForm struct {
	TaskType        int    `form:"task_type"`
	ProductCategory int    `form:"product_category"`
	ProductVersion  int    `form:"product_version"`
	OperatingSystem int    `form:"operating_system"`
	TaskSeverity    int    `form:"task_severity"`
	TaskPriority    int    `form:"task_priority"`
	ClosedByVersion int    `form:"closed_by_version"`
	ItemStatus      int    `form:"item_status"`
	ItemSummary     string `form:"item_summary"`
	DetailedDesc    string `form:"detailed_desc"`
	Action          string `form:"action"`
	ProjectId       int    `form:"project_id"`
}

// Default new task form
func DefaultNewTaskForm() *NewTaskForm {
	return &NewTaskForm{
		TaskType:        1,                 // Default bugs
		ProductCategory: 1,                 // Default Backend / Core
		ProductVersion:  1,                 // Default Development
		OperatingSystem: 1,                 // Default All
		TaskSeverity:    2,                 // Default Low
		TaskPriority:    2,                 // Default Normal
		ClosedByVersion: 0,                 // Default none
		ItemStatus:      2,                 // Default New
		ItemSummary:     "",                // Default empty
		DetailedDesc:    "",                // Default empty
		Action:          "newtask.newtask", // fixed value for new tasks
		ProjectId:       1,                 // Default project
	}
}

// Post a new task on flyspray
func (s *FSSession) NewTask(newTaskForm *NewTaskForm) error {

	// Set up query url
	queryURL := s.endpoint.urlFor(newtaskEndpoint)

	form := structToMap(newTaskForm)

	// Create the http request
	request, err := http.NewRequest("POST", queryURL, bytes.NewReader([]byte(form.Encode())))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Setup cookies
	for _, c := range s.cookies {
		request.AddCookie(c)
	}

	// Execute the request
	resp, err := s.endpoint.Client.RoundTrip(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return errors.New("Failed to create task, bad status")
	}

	s.cookies = resp.Cookies()
	return nil
}

func structToMap(i interface{}) (values url.Values) {
	values = url.Values{}
	iVal := reflect.ValueOf(i).Elem()
	typ := iVal.Type()
	for i := 0; i < iVal.NumField(); i++ {
		f := iVal.Field(i)
		tag := typ.Field(i).Tag.Get("form")
		if tag == "" {
			tag = typ.Field(i).Name
		}
		// Convert each type into a string for the url.Values string map
		var v string
		switch f.Interface().(type) {
		case int, int8, int16, int32, int64:
			v = strconv.FormatInt(f.Int(), 10)
		case uint, uint8, uint16, uint32, uint64:
			v = strconv.FormatUint(f.Uint(), 10)
		case float32:
			v = strconv.FormatFloat(f.Float(), 'f', 4, 32)
		case float64:
			v = strconv.FormatFloat(f.Float(), 'f', 4, 64)
		case []byte:
			v = string(f.Bytes())
		case string:
			v = f.String()
		}
		values.Set(tag, v)
	}
	return
}
