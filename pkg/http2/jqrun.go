package http2

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
)

// funcNames을 실제 JQ함수 배열로 변환
// 예) ko2num, ....
// namedJQFunctions는 사용자 정의 JQ함수들을 전달 가능 (없으면 DefaultJQFunctions)
// func mapToJQFunctions(funcNames []string, namedJQFunctions ...NamedJQFunctions) []gojq.CompilerOption {
// 	var userJQFunctions NamedJQFunctions = DefaultJQFunctions
// 	if len(namedJQFunctions) > 0 {
// 		userJQFunctions = namedJQFunctions[0]
// 	}

// 	var options []gojq.CompilerOption
// 	for _, funcName := range funcNames {
// 		function, ok := userJQFunctions[funcName]
// 		if !ok {
// 			return nil
// 		}
// 		option := gojq.WithFunction(funcName, 0, 0, function)
// 		options = append(options, option)
// 	}
// 	return options
// }

type JQFunction struct { // JQ함수 하나
	Name     string                    // JQ함수 이름
	MinArity int                       // JQ함수 최소 인자 개수 (default: 0)
	MaxArity int                       // JQ함수 최대 인자 개수 (default: 0)
	Function func(x any, xs []any) any // JQ함수 실제 구현
}

type JQFunctions struct { // JQ함수 배열
	Functions []JQFunction // JQ함수 배열 (없으면 DefaultJQFunctions를 사용)
}

func (j *JQFunction) ToCompilerOption() gojq.CompilerOption {
	return gojq.WithFunction(j.Name, j.MinArity, j.MaxArity, j.Function)
}

func (j *JQFunctions) ToCompilerOptions() []gojq.CompilerOption {
	var options []gojq.CompilerOption
	for _, jqFunction := range j.Functions {
		options = append(options, jqFunction.ToCompilerOption())
	}
	return options
}

// JQRun은 JQ함수를 실행합니다.
// input: JSON 문자열, filter: JQ 필터
// jqFuncs: JQ함수들을 전달 가능 (DefaultJQFunctions 참고)
// 참고: https://github.com/argoproj/argo-workflows/blob/0fd42276b827e63e6f36e0f2dcb88b6b7a959765/workflow/executor/resource.go#L15
func JQRun(input, filter string, jqFuncs ...JQFunctions) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return "", err
	}
	q, err := gojq.Parse(filter)
	if err != nil {
		return "", err
	}

	var iter gojq.Iter
	if len(jqFuncs) > 0 {
		// "ko2num"와 같은 임의의 사용자함수(for jq) code객체 추가 가능
		options := jqFuncs[0].ToCompilerOptions()
		code, err2 := gojq.Compile(q, options...)
		if err2 != nil {
			return "", err2
		}
		iter = code.Run(v)
	} else {
		iter = q.Run(v)
	}

	var buf strings.Builder
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			fmt.Println(err)
			return "", err
		}
		if s, ok := v.(string); ok {
			buf.WriteString(s)
		} else {
			b, err := json.Marshal(v)
			if err != nil {
				fmt.Println(err)
				return "", err
			}
			buf.Write(b)
		}
		buf.WriteString("\n")
	}
	return strings.TrimSpace(buf.String()), nil
}
