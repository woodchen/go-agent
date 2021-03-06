package test

import (
	"errors"
	"testing"

	"github.com/newrelic/go-agent/api"
	ats "github.com/newrelic/go-agent/api/attributes"
	"github.com/newrelic/go-agent/internal"
)

func TestUserAttributeBasics(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)

	txn.NoticeError(errors.New("zap"))

	if err := txn.AddAttribute(`int\key`, 1); nil != err {
		t.Error(err)
	}
	if err := txn.AddAttribute(`str\key`, `zip\zap`); nil != err {
		t.Error(err)
	}
	err := txn.AddAttribute("invalid_value", struct{}{})
	if _, ok := err.(internal.ErrInvalidAttribute); !ok {
		t.Error(err)
	}
	txn.End()
	if err := txn.AddAttribute("already_ended", "zap"); err != internal.ErrAlreadyEnded {
		t.Error(err)
	}

	agentAttributes := map[string]interface{}{}
	userAttributes := map[string]interface{}{`int\key`: 1, `str\key`: `zip\zap`}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "OtherTransaction/Go/hello",
		Zone:            "",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestUserAttributeBasics",
		URL:             "",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestUserAttributeConfiguration(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.TransactionEvents.Attributes.Exclude = []string{"only_errors"}
		cfg.ErrorCollector.Attributes.Exclude = []string{"only_txn_events"}
		cfg.Attributes.Exclude = []string{"completed_excluded"}
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)

	txn.NoticeError(errors.New("zap"))

	if err := txn.AddAttribute("only_errors", 1); nil != err {
		t.Error(err)
	}
	if err := txn.AddAttribute("only_txn_events", 2); nil != err {
		t.Error(err)
	}
	if err := txn.AddAttribute("completed_excluded", 3); nil != err {
		t.Error(err)
	}
	txn.End()

	agentAttributes := map[string]interface{}{}
	errorUserAttributes := map[string]interface{}{"only_errors": 1}
	txnEventUserAttributes := map[string]interface{}{"only_txn_events": 2}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "OtherTransaction/Go/hello",
		Zone:            "",
		AgentAttributes: agentAttributes,
		UserAttributes:  txnEventUserAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestUserAttributeConfiguration",
		URL:             "",
		AgentAttributes: agentAttributes,
		UserAttributes:  errorUserAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  errorUserAttributes,
	}})
}

func TestAgentAttributes(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.HostDisplayName = `my\host\display\name`
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{
		ats.HostDisplayName:              `my\host\display\name`,
		ats.ResponseCode:                 `404`,
		ats.ResponseHeadersContentType:   `text/plain; charset=us-ascii`,
		ats.ResponseHeadersContentLength: 345,
		ats.RequestMethod:                "GET",
		ats.RequestAcceptHeader:          "text/plain",
		ats.RequestContentType:           "text/html; charset=utf-8",
		ats.RequestContentLength:         753,
		ats.RequestHeadersHost:           "my_domain.com",
	}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})

	agentAttributes[ats.RequestHeadersUserAgent] = "Mozilla/5.0"
	agentAttributes[ats.RequestHeadersReferer] = "http://en.wikipedia.org/zip"

	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAgentAttributes",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestAttributesDisabled(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.Attributes.Enabled = false
		cfg.HostDisplayName = `my\host\display\name`
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.AddAttribute("my_attribute", "zip")
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAttributesDisabled",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestDefaultResponseCode(t *testing.T) {
	app := testApp(nil, nil, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))
	txn.Write([]byte("hello"))
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{
		ats.ResponseCode:         `200`,
		ats.RequestMethod:        "GET",
		ats.RequestAcceptHeader:  "text/plain",
		ats.RequestContentType:   "text/html; charset=utf-8",
		ats.RequestContentLength: 753,
		ats.RequestHeadersHost:   "my_domain.com",
	}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})

	agentAttributes[ats.RequestHeadersUserAgent] = "Mozilla/5.0"
	agentAttributes[ats.RequestHeadersReferer] = "http://en.wikipedia.org/zip"

	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestDefaultResponseCode",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestTxnEventAttributesDisabled(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.TransactionEvents.Attributes.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))
	txn.AddAttribute("myStr", "hello")
	txn.Write([]byte("hello"))
	txn.End()

	userAttributes := map[string]interface{}{
		"myStr": "hello",
	}
	agentAttributes := map[string]interface{}{
		ats.ResponseCode:         `200`,
		ats.RequestMethod:        "GET",
		ats.RequestAcceptHeader:  "text/plain",
		ats.RequestContentType:   "text/html; charset=utf-8",
		ats.RequestContentLength: 753,
		ats.RequestHeadersHost:   "my_domain.com",
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{},
	}})

	agentAttributes[ats.RequestHeadersUserAgent] = "Mozilla/5.0"
	agentAttributes[ats.RequestHeadersReferer] = "http://en.wikipedia.org/zip"

	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestTxnEventAttributesDisabled",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestErrorAttributesDisabled(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.ErrorCollector.Attributes.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))
	txn.AddAttribute("myStr", "hello")
	txn.Write([]byte("hello"))
	txn.End()

	userAttributes := map[string]interface{}{
		"myStr": "hello",
	}
	agentAttributes := map[string]interface{}{
		ats.ResponseCode:         `200`,
		ats.RequestMethod:        "GET",
		ats.RequestAcceptHeader:  "text/plain",
		ats.RequestContentType:   "text/html; charset=utf-8",
		ats.RequestContentLength: 753,
		ats.RequestHeadersHost:   "my_domain.com",
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestErrorAttributesDisabled",
		URL:             "/hello",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{},
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{},
	}})
}

var (
	allAgentAttributeNames = []string{
		ats.ResponseCode,
		ats.RequestMethod,
		ats.RequestAcceptHeader,
		ats.RequestContentType,
		ats.RequestContentLength,
		ats.RequestHeadersHost,
		ats.ResponseHeadersContentType,
		ats.ResponseHeadersContentLength,
		ats.HostDisplayName,
		ats.RequestHeadersUserAgent,
		ats.RequestHeadersReferer,
	}
)

func TestAgentAttributesExcluded(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{}

	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAgentAttributesExcluded",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestAgentAttributesExcludedFromErrors(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.ErrorCollector.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{
		ats.HostDisplayName:              `my\host\display\name`,
		ats.ResponseCode:                 `404`,
		ats.ResponseHeadersContentType:   `text/plain; charset=us-ascii`,
		ats.ResponseHeadersContentLength: 345,
		ats.RequestMethod:                "GET",
		ats.RequestAcceptHeader:          "text/plain",
		ats.RequestContentType:           "text/html; charset=utf-8",
		ats.RequestContentLength:         753,
		ats.RequestHeadersHost:           "my_domain.com",
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAgentAttributesExcludedFromErrors",
		URL:             "/hello",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  userAttributes,
	}})
}

func TestAgentAttributesExcludedFromTxnEvents(t *testing.T) {
	cfgfn := func(cfg *api.Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.TransactionEvents.Attributes.Exclude = allAgentAttributeNames
	}

	app := testApp(nil, cfgfn, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello", w, helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := txn.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	txn.WriteHeader(404)
	txn.End()

	userAttributes := map[string]interface{}{}
	agentAttributes := map[string]interface{}{
		ats.HostDisplayName:              `my\host\display\name`,
		ats.ResponseCode:                 `404`,
		ats.ResponseHeadersContentType:   `text/plain; charset=us-ascii`,
		ats.ResponseHeadersContentLength: 345,
		ats.RequestMethod:                "GET",
		ats.RequestAcceptHeader:          "text/plain",
		ats.RequestContentType:           "text/html; charset=utf-8",
		ats.RequestContentLength:         753,
		ats.RequestHeadersHost:           "my_domain.com",
		ats.RequestHeadersUserAgent:      "Mozilla/5.0",
		ats.RequestHeadersReferer:        "http://en.wikipedia.org/zip",
	}
	app.ExpectTxnEvents(t, []internal.WantTxnEvent{{
		Name:            "WebTransaction/Go/hello",
		Zone:            "F",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		Caller:          "test.TestAgentAttributesExcludedFromTxnEvents",
		URL:             "/hello",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantErrorEvent{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}
