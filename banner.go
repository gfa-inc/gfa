package gfa

import (
	"github.com/common-nighthawk/go-figure"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
)

type bannerConfig struct {
	Text    string
	Enabled bool
}

func printBanner() {
	bannerLogger := logger.New(logger.Config{})
	option := bannerConfig{
		Enabled: true,
		Text:    "GFA",
	}
	err := config.UnmarshalKey("banner", &option)
	if err != nil {
		bannerLogger.Errorf(nil, "banner config error: %v", err)
		return
	}

	if !option.Enabled {
		bannerLogger.Debug(nil, "banner is disabled")
		return
	}

	name := config.GetString("name")
	if name != "" {
		option.Text = name
	}
	figure.NewFigure(option.Text, "", true).Print()
}
