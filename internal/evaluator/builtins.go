package evaluator

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"os"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/uuid"
)

func now(args ...ref.Val) ref.Val {
	return types.Timestamp{Time: time.Now()}
}

func today(args ...ref.Val) ref.Val {
	n := time.Now()
	t := time.Date(
		n.Year(),
		n.Month(),
		n.Day(),
		0, 0, 0, 0,
		n.Location(),
	)

	return types.Timestamp{Time: t}
}

func yesterday(args ...ref.Val) ref.Val {
	return types.Timestamp{
		Time: time.Now().AddDate(0, 0, -1),
	}
}

func tomorrow(args ...ref.Val) ref.Val {
	return types.Timestamp{
		Time: time.Now().AddDate(0, 0, 1),
	}
}

func addDays(args ...ref.Val) ref.Val {
	t := args[0].(types.Timestamp)
	days := int(args[1].(types.Int))

	return types.Timestamp{
		Time: t.Time.AddDate(0, 0, days),
	}
}

func addMonths(args ...ref.Val) ref.Val {
	t := args[0].(types.Timestamp)
	months := int(args[1].(types.Int))

	return types.Timestamp{
		Time: t.Time.AddDate(0, months, 0),
	}
}

func addYears(args ...ref.Val) ref.Val {
	t := args[0].(types.Timestamp)
	years := int(args[1].(types.Int))

	return types.Timestamp{
		Time: t.Time.AddDate(years, 0, 0),
	}
}

func addDuration(args ...ref.Val) ref.Val {
	t := args[0].(types.Timestamp)

	d, err := time.ParseDuration(string(args[1].(types.String)))
	if err != nil {
		return types.NewErr(err.Error())
	}

	return types.Timestamp{
		Time: t.Time.Add(d),
	}
}

func formatDate(args ...ref.Val) ref.Val {
	t := args[0].(types.Timestamp)
	layout := string(args[1].(types.String))

	return types.String(t.Time.Format(layout))
}

func unix(args ...ref.Val) ref.Val {
	return types.Int(args[0].(types.Timestamp).Time.Unix())
}

func unixMilli(args ...ref.Val) ref.Val {
	return types.Int(args[0].(types.Timestamp).Time.UnixMilli())
}

func uuidFunc(args ...ref.Val) ref.Val {
	return types.String(uuid.NewString())
}

func envFunc(args ...ref.Val) ref.Val {
	name := string(args[0].(types.String))
	return types.String(os.Getenv(name))
}

func sha256Func(args ...ref.Val) ref.Val {
	s := string(args[0].(types.String))
	sum := sha256.Sum256([]byte(s))
	return types.String(hex.EncodeToString(sum[:]))
}

func md5Func(args ...ref.Val) ref.Val {
	s := string(args[0].(types.String))
	sum := md5.Sum([]byte(s))
	return types.String(hex.EncodeToString(sum[:]))
}

func base64Func(args ...ref.Val) ref.Val {
	s := string(args[0].(types.String))
	return types.String(base64.StdEncoding.EncodeToString([]byte(s)))
}

func urlEncode(args ...ref.Val) ref.Val {
	s := string(args[0].(types.String))
	return types.String(url.QueryEscape(s))
}

func Builtins() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Function(
			"now",
			cel.Overload(
				"now",
				nil,
				cel.TimestampType,
				cel.FunctionBinding(now),
			),
		),

		cel.Function(
			"today",
			cel.Overload(
				"today",
				nil,
				cel.TimestampType,
				cel.FunctionBinding(today),
			),
		),

		cel.Function(
			"yesterday",
			cel.Overload(
				"yesterday",
				nil,
				cel.TimestampType,
				cel.FunctionBinding(yesterday),
			),
		),

		cel.Function(
			"tomorrow",
			cel.Overload(
				"tomorrow",
				nil,
				cel.TimestampType,
				cel.FunctionBinding(tomorrow),
			),
		),

		cel.Function(
			"addDays",
			cel.Overload(
				"add_days_timestamp",
				[]*cel.Type{
					cel.TimestampType,
					cel.IntType,
				},
				cel.TimestampType,
				cel.FunctionBinding(addDays),
			),
		),

		cel.Function(
			"addMonths",
			cel.Overload(
				"add_months_timestamp",
				[]*cel.Type{
					cel.TimestampType,
					cel.IntType,
				},
				cel.TimestampType,
				cel.FunctionBinding(addMonths),
			),
		),

		cel.Function(
			"addYears",
			cel.Overload(
				"add_years_timestamp",
				[]*cel.Type{
					cel.TimestampType,
					cel.IntType,
				},
				cel.TimestampType,
				cel.FunctionBinding(addYears),
			),
		),

		cel.Function(
			"addDuration",
			cel.Overload(
				"add_duration_timestamp",
				[]*cel.Type{
					cel.TimestampType,
					cel.StringType,
				},
				cel.TimestampType,
				cel.FunctionBinding(addDuration),
			),
		),

		cel.Function(
			"formatDate",
			cel.Overload(
				"format_date",
				[]*cel.Type{
					cel.TimestampType,
					cel.StringType,
				},
				cel.StringType,
				cel.FunctionBinding(formatDate),
			),
		),

		cel.Function(
			"unix",
			cel.Overload(
				"unix",
				[]*cel.Type{
					cel.TimestampType,
				},
				cel.IntType,
				cel.FunctionBinding(unix),
			),
		),

		cel.Function(
			"unixMilli",
			cel.Overload(
				"unix_milli",
				[]*cel.Type{
					cel.TimestampType,
				},
				cel.IntType,
				cel.FunctionBinding(unixMilli),
			),
		),

		cel.Function(
			"uuid",
			cel.Overload(
				"uuid",
				nil,
				cel.StringType,
				cel.FunctionBinding(uuidFunc),
			),
		),

		cel.Function(
			"env",
			cel.Overload(
				"env",
				[]*cel.Type{
					cel.StringType,
				},
				cel.StringType,
				cel.FunctionBinding(envFunc),
			),
		),

		cel.Function(
			"sha256",
			cel.Overload(
				"sha256",
				[]*cel.Type{
					cel.StringType,
				},
				cel.StringType,
				cel.FunctionBinding(sha256Func),
			),
		),

		cel.Function(
			"md5",
			cel.Overload(
				"md5",
				[]*cel.Type{
					cel.StringType,
				},
				cel.StringType,
				cel.FunctionBinding(md5Func),
			),
		),

		cel.Function(
			"base64",
			cel.Overload(
				"base64",
				[]*cel.Type{
					cel.StringType,
				},
				cel.StringType,
				cel.FunctionBinding(base64Func),
			),
		),

		cel.Function(
			"urlEncode",
			cel.Overload(
				"url_encode",
				[]*cel.Type{
					cel.StringType,
				},
				cel.StringType,
				cel.FunctionBinding(urlEncode),
			),
		),
	}
}
