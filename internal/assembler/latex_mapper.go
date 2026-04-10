package assembler

import (
	"strings"

	"github.com/tidwall/gjson"
)

func MapParsedToMacros(parsed gjson.Result) (string, error) {
	var builder strings.Builder
	parseJSONNode("", parsed, &builder)
	return builder.String(), nil
}

func MapToMacros(jsonPayload []byte) (string, error) {
	parsed := gjson.ParseBytes(jsonPayload)
	return MapParsedToMacros(parsed)
}

func parseJSONNode(prefix string, node gjson.Result, builder *strings.Builder) {
	if node.IsObject() {
		node.ForEach(func(key, value gjson.Result) bool {
			newPrefix := makeKey(prefix, key.String())
			parseJSONNode(newPrefix, value, builder)
			return true
		})
	} else {
		builder.WriteString("\\def\\" + prefix + "{" + escapeLaTeX(node.String()) + "}\n")
	}
}

func makeKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + strings.ToUpper(key[:1]) + key[1:]
}

var latexEscaper = strings.NewReplacer(
	"%", "\\%", "$", "\\$", "&", "\\&", "#", "\\#",
	"_", "\\_", "{", "\\{", "}", "\\}",
)

func escapeLaTeX(val string) string {
	return latexEscaper.Replace(val)
}
