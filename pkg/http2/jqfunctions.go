package http2

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var DefaultJQFunctions JQFunctions = JQFunctions{
	Functions: []JQFunction{
		{Name: "ko2num", MinArity: 0, MaxArity: 0, Function: ko2num},
	},
}

// doc, _ = JQRun("2,300억", ".|ko2num", DefaultJQFunctions)
func ko2num(x any, xs []any) any {
	y, err := koreanNumberToInt(x.(string))
	if err != nil {
		fmt.Println(err)
	}
	return y
}

// func addLimit(x any, xs []any) any {
// 	y := x.(map[string]interface{}) //x(입력값)을 map으로 변환
// 	cd := y["cd"].(string)
// 	mk := y["mk"].(string)
// 	pcv := y["cv"].(int) - y["cq"].(int)
// 	llv, ulv := limitPrice(mk, pcv)
// 	y["llv"] = llv
// 	y["ulv"] = ulv
// 	return y
// }

func koreanNumberToInt(str string) (int64, error) {
	// 쉼표 제거
	str = strings.ReplaceAll(str, ",", "")

	var result int64 = 0

	// 조 단위 처리
	if strings.Contains(str, "조") {
		parts := strings.Split(str, "조")
		cho, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return 0, errors.New("조 단위 변환 실패: " + err.Error())
		}
		result += cho * 1_0000_0000_0000  // 조(10^12)
		str = strings.TrimSpace(parts[1]) // 조 이하 부분으로 변경
	}

	// 억 단위 처리
	if strings.Contains(str, "억") {
		parts := strings.Split(str, "억")
		eok, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return 0, errors.New("억 단위 변환 실패: " + err.Error())
		}
		result += eok * 1_0000_0000       // 억(10^8)
		str = strings.TrimSpace(parts[1]) // 억 이하 부분으로 변경
	}

	// 백만 단위 처리
	if strings.Contains(str, "백만") {
		parts := strings.Split(str, "백만")
		million, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			return 0, errors.New("백만 단위 변환 실패: " + err.Error())
		}
		result += million * 1_000_000     // 백만(10^6)
		str = strings.TrimSpace(parts[1]) // 백만 이하 부분으로 변경
	}

	// 남은 부분(만 단위 이하) 처리
	if str != "" {
		remain, err := strconv.ParseInt(strings.TrimSpace(str), 10, 64)
		if err != nil {
			return 0, errors.New("일반 숫자 변환 실패: " + err.Error())
		}
		result += remain
	}

	return result, nil
}
