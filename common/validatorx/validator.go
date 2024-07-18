package validatorx

import (
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhtrans "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
	"strings"
)

var (
	uni   *ut.UniversalTranslator
	Trans ut.Translator
)

func Setup() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		enLoc := en.New()
		zhLoc := zh.New()
		uni = ut.New(enLoc, zhLoc)
		Trans, _ = uni.GetTranslator("zh")
		err := zhtrans.RegisterDefaultTranslations(v, Trans)
		if err != nil {
			logger.Panic(err)
		}

		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			tagValue := fld.Tag.Get("json")
			if tagValue == "" {
				tagValue = fld.Tag.Get("form")
			}
			name := strings.SplitN(tagValue, ",", 2)[0]

			if name == "-" {
				return ""
			}

			return name
		})
	}
}
