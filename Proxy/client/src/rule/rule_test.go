package proxy

import "testing"

func testMatch(pattern string, content string, ans string) bool {
	ruler := NewGFWRuleParser()

	isMatched := ruler.isMatch(pattern, content)
	return isMatched == (ans == "true")
}

func testMatchRegex(pattern string, content string, ans string) bool {
	ruler := NewGFWRuleParser()

	isMatched := ruler.isRegexMatch(pattern, content)
	return isMatched == (ans == "true")
}

func TestGFWURLMatch(t *testing.T) {
	var c chan bool = make(chan bool)
	a := NewGFWRuleParser()
	a.DownloadGFWRuleListFromIntenet(func(err error) {
		a.ParseFile()
		if a.IsURLMatched("https://www.google.com") == false {
			t.Error("google你TM都不支持？")
		}
		if a.IsURLMatched("https://www.facebook.com") == false {
			t.Error("face你TM都不支持？")
		}
		if a.IsURLMatched("https://windyx.com") == true {
			t.Error("我的网站你都屏蔽？")
		}
		if a.IsURLMatched("https://www.altrec.com") == false {
			t.Error("error")
		}
		c <- true
	})
	<-c
}

func TestMatchChars(t *testing.T) {
	matchCaseList := [][]string{
		[]string{"^", "http://example.com", "true"},
		[]string{"*", "http://example.com", "true"},
		[]string{"e.", "http://example.com", "true"},
		[]string{"example.com", "http://example.com", "true"},
		[]string{"*.com", "http://example.com", "true"},
		[]string{"*le.com", "http://example.com", "true"},
		[]string{"*x*l.com", "http://example.com", "false"},
		[]string{"*x*le.com", "http://example.com", "true"},
		[]string{"*x*le.com", "http://example.com", "true"},
		[]string{"^example.com", "http://example.com", "true"},
		[]string{"^example.com^", "http://example.com.cn", "false"},
		[]string{"^example.com^", "http://example.com/cn", "true"},
		[]string{"^example.com^*n", "http://example.com/cn", "true"},
	}
	for _, testCase := range matchCaseList {
		if !testMatch(testCase[0], testCase[1], testCase[2]) {
			t.Error("\np: ", testCase[0], "\nm: ", testCase[1], "\nans: ", testCase[2], "failed")
		}
	}
}

func TestMatchBeginEndWith(t *testing.T) {
	matchCaseList := [][]string{
		[]string{"|http://example.com", "http://example.com", "true"},
		[]string{"|http://example.com", "http://example.com.cn", "true"},
		[]string{"|http://example.com", "http://example.cn", "false"},
		[]string{"http://example.com|", "http://example.com", "true"},
		[]string{"http://example.com|", "http://example.com.cn", "false"},
		[]string{"|http://example.com|", "http://example.com", "true"},
		[]string{"|http://example.com|", "http://example.com.cn", "false"},
	}
	for _, testCase := range matchCaseList {
		if !testMatch(testCase[0], testCase[1], testCase[2]) {
			t.Error("\np: ", testCase[0], "\nm: ", testCase[1], "\nans: ", testCase[2], "failed")
		}
	}
}

func TestMatchDomain(t *testing.T) {
	matchCaseList := [][]string{
		[]string{"||example.com", "http://example.com", "true"},
		[]string{"||example.com", "http://example1.com", "false"},
		[]string{"||example.com", "http://work.example.com", "true"},
		[]string{"||example.com", "http://pre.work.example.com", "true"},
		[]string{"||example.com.cn", "http://pre.work.example.com", "false"},
	}
	for _, testCase := range matchCaseList {
		if !testMatch(testCase[0], testCase[1], testCase[2]) {
			t.Error("\np: ", testCase[0], "\nm: ", testCase[1], "\nans: ", testCase[2], "failed")
		}
	}
}

func TestMatchRegex(t *testing.T) {
	matchCaseList := [][]string{
		[]string{"/", "http://example.com", "true"},
	}
	for _, testCase := range matchCaseList {
		if !testMatchRegex(testCase[0], testCase[1], testCase[2]) {
			t.Error("\np: ", testCase[0], "\nm: ", testCase[1], "\nans: ", testCase[2], "failed")
		}
	}
}
