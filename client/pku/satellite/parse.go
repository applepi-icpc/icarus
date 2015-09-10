package pku

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	tokenClassBegin = `<td class="datagrid"><a href="/elective2008/edu/pku/stu/elective/controller/supplement/goNested.do?`

	reName    = regexp.MustCompile(`(?i)<span>(.*?)</span>`)
	reThGr    = regexp.MustCompile(`(?i)<td class="datagrid"><span style="width: 90">(.*?)</span></td>\s*<td class="datagrid" align="center"><span style="width: 30">(.*?)</span>`)
	reUbound  = regexp.MustCompile(`(?i)<span id="electedNum\d*?" style="width: 60">(\d+?) / \d+?`)
	reTime    = regexp.MustCompile(`(?i)<td class="datagrid"><span style="width: 200">(.*?)</span>`)
	reCommand = regexp.MustCompile(`(?i)<td class="datagrid" align="center"><a (.)`)
	reElect   = regexp.MustCompile(`(?i)"/elective2008/edu/pku/stu/elective/controller/supplement/electSupplement.do\?index=(.*?)&amp;seq=(.*?)"`)
	reRefresh = regexp.MustCompile(`(?i)javascript:refreshLimit\('.*?','.*?','.*?','(.*?)','(.*?)','.*?'\);`)
	rePage    = regexp.MustCompile(`(?i)<td colspan="10" valign="baseline">Page \d+? of (\d+?)`)
)

// Translated from nzk-elect.
type PKUClass struct {
	Name    string
	Teacher string
	GroupID string
	UBound  int
	Msg     string
	Index   string
	Seq     string
	IsFull  bool
}

// Translated from nzk-elect.
func wrongFormatException(code int, msg string) error {
	return errors.New(fmt.Sprintf("Format error %d: %s", code, msg))
}

func parseTotalPage(s string) (int, error) {
	match := rePage.FindStringSubmatch(s)
	if match == nil {
		return 0, wrongFormatException(5, "Error when parsing total pages")
	}
	res, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}
	return res, nil
}

func parseClass(s string) (res PKUClass, err error) {
	err = nil

	match := reName.FindStringSubmatch(s)
	if match == nil {
		err = wrongFormatException(1, "Error when parsing class name")
		return
	}
	res.Name = match[1]

	match = reThGr.FindStringSubmatch(s)
	if match == nil {
		err = wrongFormatException(2, "Error when parsing teacher name and group id")
		return
	}
	res.Teacher = match[1]
	res.GroupID = match[2]

	match = reUbound.FindStringSubmatch(s)
	if match == nil {
		err = wrongFormatException(6, "Error when parsing upper bound")
		return
	}
	res.UBound, err = strconv.Atoi(match[1])
	if err != nil {
		return
	}

	match = reTime.FindStringSubmatch(s)
	if match == nil {
		err = wrongFormatException(7, "Error when parsing message")
		return
	}
	res.Msg = strings.Replace(match[1], "<br>", " ", -1)

	match = reCommand.FindStringSubmatch(s)
	if match == nil {
		err = wrongFormatException(3, "Error when checking status")
		return
	}

	var reIS *regexp.Regexp
	if match[1] == "h" {
		res.IsFull = false
		reIS = reElect
	} else {
		res.IsFull = true
		reIS = reRefresh
	}
	match = reIS.FindStringSubmatch(s)
	if match == nil {
		err = wrongFormatException(4, "Error when extracting index and sequence number.")
		return
	}
	res.Index = match[1]
	res.Seq = match[2]

	return
}

func parseList(s string) (res []PKUClass, err error) {
	res = make([]PKUClass, 0)

	p := strings.Index(s, tokenClassBegin)
	for p != -1 {
		var cl PKUClass
		cl, err = parseClass(s[p:])
		if err != nil {
			return
		}
		res = append(res, cl)
		s = s[p+1:]
		p = strings.Index(s, tokenClassBegin)
		e := strings.Index(s, "datagrid-footer")
		if p >= e {
			break
		}
	}

	return
}

func getOriginalPage(pagenum int, jsessionid string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/elective2008/edu/pku/stu/elective/controller/supplement/supplement.jsp?netui_pagesize=electableListGrid%%3B20&netui_row=electableListGrid%%3B%d", electRoot, pagenum*20), strings.NewReader(""))
	req.Header.Set("Cookie", fmt.Sprintf("JSESSIONID=%s", jsessionid))
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error requesting supplement page: %s", err.Error()))
	}
	defer res.Body.Close()
	rawResBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error reading supplement page: %s", err.Error()))
	}
	return string(rawResBody), nil
}