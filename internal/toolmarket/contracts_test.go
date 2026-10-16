package toolmarket

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestHumanProgressDoesNotSkipOrReopen(t *testing.T) {
	require.True(t, Transition("SUBMITTED", "EVALUATING"))
	require.True(t, Transition("EVALUATING", "EVALUATING"))
	require.True(t, Transition("DEVELOPING", "DELIVERED"))
	require.True(t, Transition("SUBMITTED", "CLOSED"))
	require.False(t, Transition("SUBMITTED", "DELIVERED"))
	require.False(t, Transition("DELIVERED", "DEVELOPING"))
	require.False(t, Transition("CLOSED", "CLOSED"))
	require.False(t, Transition("invented", "CLOSED"))
}
func TestDemandIsBoundedData(t *testing.T) {
	require.True(t, completeDemand().Valid())
	require.False(t, (Demand{Kind: "CODE", Title: "x", Description: "x"}).Valid())
	require.False(t, (Demand{Kind: "DATA", Title: strings.Repeat("中", 121), Description: "x"}).Valid())
	require.False(t, (Demand{Kind: "DATA", Title: "x\n", Description: "x"}).Valid())
	require.False(t, (Progress{Stage: "DELIVERED", Note: ""}).Valid())
}

func TestFullDemandRequiresContactConsentAndBoundedReferenceFiles(t *testing.T) {
	// The previous three-field payload must no longer enter the current contract.
	require.False(t, (Demand{Kind: "DATA", Title: "采集需求", Description: "保存商品资料"}).Valid())
	for _, change := range []func(*Demand){
		func(d *Demand) { d.Scenario = "" }, func(d *Demand) { d.ContactName = "" }, func(d *Demand) { d.ContactMethod = "" }, func(d *Demand) { d.Consent = false },
		func(d *Demand) { d.Files = []Upload{{Name: "../secret.txt", Data: []byte("text")}} },
		func(d *Demand) {
			d.Files = []Upload{{Name: "page.html", Data: []byte("<html><script>alert(1)</script></html>")}}
		},
		func(d *Demand) {
			d.Files = []Upload{{Name: "large.txt", Data: []byte(strings.Repeat("x", MaxFileBytes+1))}}
		},
	} {
		d := completeDemand()
		change(&d)
		require.False(t, d.Valid())
	}
	d := completeDemand()
	d.Files = []Upload{{Name: "流程.txt", Data: []byte("有界参考资料")}}
	require.True(t, d.Valid())
}
func completeDemand() Demand {
	return Demand{Kind: "DATA", Title: "采集需求", Description: "保存已授权的商品资料", Scenario: "商品管理", ContactName: "测试联系人", ContactMethod: "test-wechat", Consent: true}
}
