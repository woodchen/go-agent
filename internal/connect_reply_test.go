package internal

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCreateFullTxnNameBasic(t *testing.T) {
	emptyReply := connectReplyDefaults()

	tcs := []struct {
		input      string
		background bool
		expect     string
	}{
		{"", true, "WebTransaction/Go/"},
		{"/", true, "WebTransaction/Go/"},
		{"hello", true, "WebTransaction/Go/hello"},
		{"/hello", true, "WebTransaction/Go/hello"},

		{"", false, "OtherTransaction/Go/"},
		{"/", false, "OtherTransaction/Go/"},
		{"hello", false, "OtherTransaction/Go/hello"},
		{"/hello", false, "OtherTransaction/Go/hello"},
	}

	for _, tc := range tcs {
		if out := CreateFullTxnName(tc.input, emptyReply, tc.background); out != tc.expect {
			t.Error(tc.input, tc.background, out, tc.expect)
		}
	}
}

func TestCreateFullTxnNameURLRulesIgnore(t *testing.T) {
	js := `[{
		"match_expression":".*zip.*$",
		"ignore":true
	}]`
	reply := connectReplyDefaults()
	err := json.Unmarshal([]byte(js), &reply.URLRules)
	if nil != err {
		t.Fatal(err)
	}
	if out := CreateFullTxnName("/zap/zip/zep", reply, true); out != "" {
		t.Error(out)
	}
}

func TestCreateFullTxnNameTxnRulesIgnore(t *testing.T) {
	js := `[{
		"match_expression":"^WebTransaction/Go/zap/zip/zep$",
		"ignore":true
	}]`
	reply := connectReplyDefaults()
	err := json.Unmarshal([]byte(js), &reply.TxnNameRules)
	if nil != err {
		t.Fatal(err)
	}
	if out := CreateFullTxnName("/zap/zip/zep", reply, true); out != "" {
		t.Error(out)
	}
}

func TestCreateFullTxnNameAllRules(t *testing.T) {
	js := `{
		"url_rules":[
			{"match_expression":"zip","each_segment":true,"replacement":"zoop"}
		],
		"transaction_name_rules":[
			{"match_expression":"WebTransaction/Go/zap/zoop/zep",
			 "replacement":"WebTransaction/Go/zap/zoop/zep/zup/zyp"}
		],
		"transaction_segment_terms":[
			{"prefix": "WebTransaction/Go/",
			 "terms": ["zyp", "zoop", "zap"]}
		]
	}`
	reply := connectReplyDefaults()
	err := json.Unmarshal([]byte(js), &reply)
	if nil != err {
		t.Fatal(err)
	}
	if out := CreateFullTxnName("/zap/zip/zep", reply, true); out != "WebTransaction/Go/zap/zoop/*/zyp" {
		t.Error(out)
	}
}

func TestCalculateApdexThreshold(t *testing.T) {
	reply := connectReplyDefaults()
	threshold := calculateApdexThreshold(reply, "WebTransaction/Go/hello")
	if threshold != 500*time.Millisecond {
		t.Error("default apdex threshold", threshold)
	}

	reply = connectReplyDefaults()
	reply.ApdexThresholdSeconds = 1.3
	reply.KeyTxnApdex = map[string]float64{
		"WebTransaction/Go/zip": 2.2,
		"WebTransaction/Go/zap": 2.3,
	}
	threshold = calculateApdexThreshold(reply, "WebTransaction/Go/hello")
	if threshold != 1300*time.Millisecond {
		t.Error(threshold)
	}
	threshold = calculateApdexThreshold(reply, "WebTransaction/Go/zip")
	if threshold != 2200*time.Millisecond {
		t.Error(threshold)
	}
}
