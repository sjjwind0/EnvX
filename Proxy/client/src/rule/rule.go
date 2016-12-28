package rule

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

const gfwListURL = "https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt"
const gfwListFilePath = "/tmp/gfwList.txt"

const (
	kMatchType_BeginWith = 2 << 0
	kMatchType_EndWith   = 2 << 1
	kMatchType_Domain    = 2 << 2
	kMatchType_URLKey    = 2 << 3
)

type gfwMatchInfo struct {
}

type GFWRuleParser struct {
	ignoreRuleList []string
	adpRuleList    []string
	regexRuleList  []string
}

func NewGFWRuleParser() *GFWRuleParser {
	return new(GFWRuleParser)
}

func (g *GFWRuleParser) isGFWFileExist() bool {
	_, err := os.Stat(gfwListFilePath)
	return err == nil || os.IsExist(err)
}

func (g *GFWRuleParser) DownloadGFWRuleListFromIntenet(cb func(err error)) {
	go func() {
		if g.isGFWFileExist() {
			fmt.Println("is eixt")
			cb(nil)
			return
		}
		rsp, err := http.Get(gfwListURL)
		if err != nil {
			fmt.Println("GFWRuleParser DownloadGFWRuleListFromIntenet failed: ", err)
			cb(err)
			return
		}
		defer rsp.Body.Close()
		body, err := ioutil.ReadAll(rsp.Body)

		if err != nil {
			fmt.Println("GFWRuleParser DownloadGFWRuleListFromIntenet ReadAll failed: ", err)
			cb(err)
			return
		}

		// base64 decode
		decodeData, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			fmt.Println("GFWRuleParser DownloadGFWRuleListFromIntenet base64 decode error: ", err)
			cb(err)
			return
		}
		file, err := os.Create(gfwListFilePath)
		if err != nil {
			fmt.Println("GFWRuleParser DownloadGFWRuleListFromIntenet open file error: ", err)
			cb(err)
			return
		}
		defer file.Close()
		file.Write(decodeData)
		cb(nil)
	}()
}

func (g *GFWRuleParser) ParseFile() {
	file, err := os.Open(gfwListFilePath)
	if err != nil {
		fmt.Println("GFWRuleParser ParseFile failed: ", err)
		return
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("GFWRuleParser stat failed: ", err)
		return
	}
	var fileData []byte = make([]byte, fileInfo.Size())
	file.Read(fileData)
	g.parseData(fileData)
}

func (g *GFWRuleParser) parseData(gfwFileData []byte) error {
	var gfwFileContent string = string(gfwFileData)
	fileLines := strings.Split(gfwFileContent, "\n")
	fmt.Println("gfwFile file line count: ", len(fileLines))
	for _, line := range fileLines {
		if len(line) <= 1 || line[0] == '[' {
			continue
		}
		switch line[0] {
		case '.':
			g.adpRuleList = append(g.adpRuleList, "*"+line)
		case '!':
			continue
			// comment
		case '@':
			if line[1] == '@' {
				g.ignoreRuleList = append(g.ignoreRuleList, line[2:])
			}
		case '|':
			g.adpRuleList = append(g.adpRuleList, line)
		case '/':
			// regex
			lineLength := len(line)
			if line[lineLength-1] == '/' {
				g.regexRuleList = append(g.regexRuleList, line[1:lineLength-1])
			}
		default:
			g.adpRuleList = append(g.adpRuleList, line)
		}
	}
	return nil
}

func (g *GFWRuleParser) IsURLMatched(url string) bool {
	for _, pattern := range g.adpRuleList {
		if g.isMatch(pattern, url) {
			return true
		}
	}
	for _, pattern := range g.regexRuleList {
		if g.isMatch(pattern, url) {
			return true
		}
	}
	return false
}

func (g *GFWRuleParser) IsURLIgnored(url string) bool {
	for _, pattern := range g.ignoreRuleList {
		if g.isIgnored(pattern, url) {
			return true
		}
	}
	return false

}

func (g *GFWRuleParser) isIgnored(pattern string, content string) bool {
	return g.isMatch(pattern, content)
}

func (g *GFWRuleParser) isRegexMatch(pattern string, content string) bool {
	matched, err := regexp.MatchString(pattern, content)
	if err != nil {
		return false
	}
	return matched
}

func (g *GFWRuleParser) isMatch(pattern string, content string) bool {
	var hasBeginWithChar bool = false
	var hasEndWithChar bool = false
	var needMatchDomain bool = false
	var needMatchKey bool = false

	if pattern[0] == '|' {
		if pattern[1] == '|' {
			needMatchDomain = true
		} else {
			hasBeginWithChar = true
		}
	}
	if pattern[len(pattern)-1] == '|' {
		hasEndWithChar = true
	}

	if !(needMatchDomain == true || hasBeginWithChar == true || hasEndWithChar == true) {
		needMatchKey = true
	}

	// fmt.Println("hasBeginWithChar: ", hasBeginWithChar)
	// fmt.Println("hasEndWithChar: ", hasEndWithChar)
	// fmt.Println("needMatchDomain: ", needMatchDomain)
	// fmt.Println("needMatchKey: ", needMatchKey)

	if hasBeginWithChar {
		if hasEndWithChar {
			if g.isEndWithMatch(pattern[1:len(pattern)-1], content) {
				return true
			}
			return false
		} else {
			if g.isBeginWithMatch(pattern[1:], content) {
				return true
			}
		}
		return false
	}
	if !hasBeginWithChar && hasEndWithChar {
		if g.isEndWithMatch(pattern[:len(pattern)-1], content) {
			return true
		}
		return false
	}
	if needMatchDomain {
		if g.isDomainMatch(pattern[2:], content) {
			return true
		}
		return false
	}
	if needMatchKey {
		if g.isURLKeyMatch(pattern, content) {
			return true
		}
		return false
	}
	return false
}

func (g *GFWRuleParser) isBeginWithMatch(pattern string, content string) bool {
	return strings.HasPrefix(content, pattern)
}

func (g *GFWRuleParser) isEndWithMatch(pattern string, content string) bool {
	return strings.HasSuffix(content, pattern)
}

func (g *GFWRuleParser) isDomainMatch(pattern string, content string) bool {
	u, err := url.Parse(content)
	if err != nil {
		fmt.Println(content, "is not regular url format")
		return false
	}
	return g.isMatch(pattern, u.Host)
}

func (g *GFWRuleParser) isURLKeyMatch(pattern string, content string) bool {
	for i := 0; i < len(content); i++ {
		if g.isStringMatch(pattern, content[i:]) {
			return true
		}
	}
	return false
}

func (g *GFWRuleParser) isStringMatch(pattern string, content string) bool {
	var paternIndex int = 0
	var contentIndex int = 0
	var contentLength int = len(content)
	var patterLength int = len(pattern)
	if patterLength > contentLength || contentLength == 0 {
		return false
	}
	for {
		if paternIndex == patterLength && contentIndex <= contentLength {
			return true
		}
		if paternIndex >= patterLength || contentIndex >= contentLength {
			return false
		}
		switch pattern[paternIndex] {
		case '*':
			// match all
			if paternIndex+1 == patterLength || contentIndex+1 >= contentLength {
				return true
			}
			for tryMatchIndex := contentIndex + 1; tryMatchIndex < contentLength; tryMatchIndex++ {
				tmpPattern := pattern[paternIndex+1:]
				tmpContent := content[tryMatchIndex:]
				if len(tmpPattern) > len(tmpContent) {
					break
				}
				if g.isStringMatch(tmpPattern, tmpContent) {
					return true
				}
			}
			return false
		case '^':
			contentChar := content[contentIndex]
			switch {
			case contentChar >= '0' && contentChar <= '9':
				fallthrough
			case contentChar >= 'a' && contentChar <= 'z':
				fallthrough
			case contentChar >= 'A' && contentChar <= 'Z':
				fallthrough
			case contentChar == '_':
				fallthrough
			case contentChar == '-':
				fallthrough
			case contentChar == '.':
				fallthrough
			case contentChar == '%':
				if paternIndex == 0 {
					contentIndex = contentIndex + 1
				} else {
					return false
				}
			default:
				contentIndex = contentIndex + 1
				paternIndex = paternIndex + 1
			}
		default:
			contentChar := content[contentIndex]
			patternChar := pattern[paternIndex]
			if contentChar != patternChar {
				if paternIndex == 0 {
					contentIndex = contentIndex + 1
				} else {
					return false
				}
			} else {
				contentIndex = contentIndex + 1
				paternIndex = paternIndex + 1
			}
		}
	}
}
