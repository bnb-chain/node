package transaction

import "encoding/json"

// Transaction represents a Dredd transaction object.
// http://dredd.readthedocs.io/en/latest/data-structures/#transaction
type Transaction struct {
	Id       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     string `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	FullPath string `json:"fullPath,omitempty"`
	Request  *struct {
		Body    string                 `json:"body,omitempty"`
		Headers map[string]interface{} `json:"headers,omitempty"`
		URI     string                 `json:"uri,omitempty"`
		Method  string                 `json:"method,omitempty"`
	} `json:"request,omitempty"`
	Expected *struct {
		StatusCode string                 `json:"statusCode,omitempty"`
		Body       string                 `json:"body,omitempty"`
		Headers    map[string]interface{} `json:"headers,omitempty"`
		Schema     *json.RawMessage       `json:"bodySchema,omitempty"`
	} `json:"expected,omitempty"`
	Real *struct {
		Body       string                 `json:"body"`
		Headers    map[string]interface{} `json:"headers"`
		StatusCode int                    `json:"statusCode"`
	} `json:"real,omitempty"`
	Origin  *json.RawMessage `json:"origin,omitempty"`
	Test    *json.RawMessage `json:"test,omitempty"`
	Results *json.RawMessage `json:"results,omitempty"`
	Skip    bool             `json:"skip"`
	Fail    interface{}      `json:"fail,omitempty"`

	TestOrder []string `json:"hooks_modifications,omitempty"`
}

// AddTestOrderPoint adds a value to the hooks_modification key used when
// running dredd with TEST_DREDD_HOOKS_HANDLER_ORDER enabled.
func (t *Transaction) AddTestOrderPoint(value string) {
	if t.TestOrder == nil {
		t.TestOrder = []string{}
	}
	t.TestOrder = append(t.TestOrder, value)
}
