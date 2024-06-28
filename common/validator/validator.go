package validator

import (
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhtrans "github.com/go-playground/validator/v10/translations/zh"
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
	}
}
